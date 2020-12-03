package clickhouse

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	// Clickhouse driver
	_ "github.com/ClickHouse/clickhouse-go"
	"github.com/archaron/juniper-natlog/common"
	"github.com/im-kulikov/helium/service"
	"github.com/spf13/viper"
	"go.uber.org/dig"
	"go.uber.org/zap"
)

type (
	Settings struct {
		Address   string
		Username  string
		Password  string
		Database  string
		FlowTable string
		Debug     bool
		BatchSize int
		BatchTimeout time.Duration

		ReadTimeout  int
		WriteTimeout int
	}

	clickhouseOutParams struct {
		dig.Out
		Service    service.Service `group:"services"`
		Clickhouse *Service
	}

	Service struct {
		con *sql.DB
		log *zap.Logger
		cfg *Settings

		once   sync.Once
		cancel context.CancelFunc

		pool   chan *common.FlowMessage
		models map[string]*common.Model
	}
)

func (s *Service) Ping() error {
	return s.con.Ping()
}

func (s *Service) Start(ctx context.Context) error {
	s.once.Do(func() {
		ctx, s.cancel = context.WithCancel(ctx)
		go s.Worker(ctx)
	})
	return nil
}

func (s *Service) Stop() error {
	s.cancel()
	if s.con != nil {
		return s.con.Close()
	}
	return nil
}

func (s *Service) Name() string {
	return "clickhouse"
}

func (s Settings) buildDSN() string {
	u := url.URL{
		Scheme: "tcp",
		Host:   s.Address,
	}

	q := u.Query()
	q.Set("username", s.Username)
	q.Set("password", s.Password)
	q.Set("database", s.Database)
	q.Set("read_timeout", strconv.Itoa(s.ReadTimeout))
	q.Set("write_timeout", strconv.Itoa(s.WriteTimeout))
	if s.Debug {
		q.Set("debug", "true")
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// newSettings - Read service settings and validate them
func newSettings(v *viper.Viper) (*Settings, error) {
	var cfg Settings

	if !v.IsSet("clickhouse") {
		return nil, errors.New("'clickhouse' section not found in config")
	}

	v.SetDefault("clickhouse.address", "127.0.0.1:9000")
	cfg.Address = v.GetString("clickhouse.address")

	v.SetDefault("clickhouse.username", "default")
	cfg.Username = v.GetString("clickhouse.username")

	v.SetDefault("clickhouse.password", "")
	cfg.Password = v.GetString("clickhouse.password")

	v.SetDefault("clickhouse.batch_size", 10000)
	cfg.BatchSize = v.GetInt("clickhouse.batch_size")

	v.SetDefault("clickhouse.batch_timeout", 60 * time.Second)
	cfg.BatchTimeout = v.GetDuration("clickhouse.batch_timeout")

	v.SetDefault("clickhouse.database", "default")
	cfg.Database = v.GetString("clickhouse.database")

	v.SetDefault("clickhouse.read_timeout", 30)
	cfg.ReadTimeout = v.GetInt("clickhouse.read_timeout")

	v.SetDefault("clickhouse.write_timeout", 30)
	cfg.WriteTimeout = v.GetInt("clickhouse.write_timeout")

	cfg.Debug = v.GetBool("clickhouse.debug")

	return &cfg, nil
}

// newService - Create service
func newService(cfg *Settings, log *zap.Logger) (clickhouseOutParams, error) {
	var (
		err error
		out clickhouseOutParams
	)

	ch := &Service{
		log:    log,
		cfg:    cfg,
		models: make(map[string]*common.Model),
		cancel: func() {},
	}

	ch.con, err = sql.Open("clickhouse", cfg.buildDSN())
	if err != nil {
		return out, err
	}

	if err = ch.con.Ping(); err != nil {
		return out, err
	}

	ch.pool = make(chan *common.FlowMessage, cfg.BatchSize)
	out.Clickhouse = ch
	out.Service = ch

	log.Info("clickhouse connected")

	return out, nil
}

func (s *Service) RegisterModel(rule string, model *common.Model) {
	s.log.Debug("register model", zap.String("rule", rule))
	if err := s.compileSQLTemplate(model); err != nil {
		s.log.Fatal("cannot compile sql statement", zap.Error(err))
	}
	s.models[rule] = model
}

func (s *Service) Insert(message *common.FlowMessage) {

	if s.pool == nil {
		s.log.Fatal("pool is nil")
	}
	s.pool <- message
}

func (s *Service) Worker(ctx context.Context) {
	s.log.Debug("batch", zap.Duration("timeout", s.cfg.BatchTimeout), zap.Int("size", s.cfg.BatchSize))
	var (
		done   = make(chan *common.PoolBump, 10)
		pool   = make(map[string]*common.PoolItem, len(s.models))
		ticker = time.NewTimer(s.cfg.BatchTimeout)
	)

	for rule := range s.models {
		pool[rule] = &common.PoolItem{
			Size:  0,
			Items: make([]common.FlowMessagePayload, 0, s.cfg.BatchSize),
			Last: time.Now(),
		}
	}

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case <-ticker.C:
			for rule := range pool {
				if pool[rule].Size > 0  && time.Since(pool[rule].Last) >= s.cfg.BatchTimeout {
					done <- &common.PoolBump{
						Reason: "ticker",
						Rule:   rule,
					}
				}
			}
			ticker.Reset(s.cfg.BatchTimeout)
		case msg := <-s.pool:
			var (
				ruleItem *common.PoolItem
				ok bool
			)

			if ruleItem, ok =pool[msg.Rule]; !ok {
				s.log.Error("unknown message rule", zap.String("rule", msg.Rule))
				continue loop
			}


			 ruleItem.Items = append(ruleItem.Items, msg.Fields)
			 ruleItem.Size += len(msg.Fields)

			if  ruleItem.Size < s.cfg.BatchSize {
				continue loop
			}
			done <- &common.PoolBump{
				Reason: "filled",
				Rule:   msg.Rule,
			}

		case bump := <-done:
			total := 0
			now := time.Now()

			ruleItem := pool[bump.Rule]


			if ruleItem.Size == 0 {
				continue loop
			}

			tx, _ := s.con.Begin()

			stmt, err := tx.Prepare(s.models[bump.Rule].Statement)
			if err != nil {
				s.log.Error("could not prepare insert statement", zap.Error(err))
				continue loop
			}

			for _, msg := range ruleItem.Items {
				values := make([]interface{}, 0, len(msg))
				for _, field := range s.models[bump.Rule].Fields {
					val, err := field.Convert(msg[field.GetName()])
					if err != nil {
						s.log.Error("cannot convert field", zap.Error(err), zap.String("rule", bump.Rule), zap.String("field", field.GetName()), zap.String("reason", bump.Reason))
						continue
					}
					values = append(values, val)
					total++
				}

				if _, err := stmt.Exec(values...); err != nil {

					s.log.Error("could not exec insert statement", zap.Error(err))
					if err = stmt.Close(); err != nil {
						s.log.Error("could not close statement", zap.Error(err))
					}

					if err = tx.Rollback(); err != nil {
						s.log.Error("transaction rollback error", zap.Error(err))
					}

					continue loop

				}
			}

			if err := tx.Commit(); err != nil {
				s.log.Error("could not commit transaction", zap.Error(err))
				if err = stmt.Close(); err != nil {
					s.log.Error("could not close statement", zap.Error(err))
				}

				continue loop

			}

			s.log.Debug("inserted", zap.String("rule",bump.Rule), zap.String("reason", bump.Reason), zap.Int("records", total), zap.Duration("time", time.Since(now)))

			ruleItem.Items = make([]common.FlowMessagePayload, 0,s.cfg.BatchSize)
			ruleItem.Size = 0
			ruleItem.Last = now


		}

	}

	ticker.Stop()
}

type insertData struct {
	Fields    []common.ConvertableField
	TableName string
}

func (s *Service) compileSQLTemplate(model *common.Model) error {
	var buf strings.Builder

	err := insertTemplate.ExecuteTemplate(&buf, "insertTemplate", &insertData{
		Fields:    model.Fields,
		TableName: model.Table,
	})

	if err != nil {
		return err
	}
	model.Statement = buf.String()
	return nil
}

var insertTemplate, _ = template.New("insertTemplate").Funcs(template.FuncMap{
	"last": func(x int, a interface{}) bool {
		return x == reflect.ValueOf(a).Len()-1
	},
}).Parse(`
	INSERT INTO {{.TableName}}
	(
		{{range  $i, $e := .Fields}}{{$e.Name}}{{if last $i $.Fields}}{{else}},{{end}}{{end}}
	) VALUES (
		{{range  $i, $e := .Fields}}?{{if last $i $.Fields}}{{else}},{{end}}{{end}}
	)
`)
