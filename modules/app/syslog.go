package app

import (
	"context"
	"reflect"
	"regexp"
	"time"

	"github.com/archaron/juniper-natlog/common"
	"github.com/archaron/juniper-natlog/modules/clickhouse"
	"github.com/im-kulikov/helium/service"
	"github.com/im-kulikov/helium/web"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"go.uber.org/dig"
	"go.uber.org/zap"
	"gopkg.in/mcuadros/go-syslog.v2"
)

type (
	syslogParams struct {
		dig.In
		Viper  *viper.Viper
		Logger *zap.Logger
		CH     *clickhouse.Service
	}

	syslogOutParams struct {
		dig.Out
		Service service.Service `group:"services"`
	}

	syslogListener struct {
		timeout time.Duration
		log     *zap.Logger

		address    string
		msgChannel syslog.LogPartsChannel
		handler    *syslog.ChannelHandler
		server     *syslog.Server
		ch         *clickhouse.Service

		rules common.Rules
	}
)

func (s *syslogListener) registerModels() {
	for _, r := range s.rules {
		model := common.Model{
			Table: r.Table,
		}

		log := s.log.With(zap.String("rule", r.Name))

		for i, f := range r.Fields {
			name, ok := f["name"].(string)
			if !ok {
				log.Fatal("field name is not specified or is not a string", zap.Int("index", i))
			}

			log := log.With(zap.Int("index", i), zap.String("name", name))

			t, ok := f["type"].(string)
			if !ok {
				log.Fatal("field typ is not specified or is not a string")
			}

			log = log.With(zap.String("type", t))

			modelField := common.ModelField{
				Name: name,
				Type: t,
			}

			switch t {
			case "string":
				model.Fields = append(model.Fields, &common.StringModelField{
					ModelField: modelField,
				})
			case "timestamp":
				layout, ok := f["default"].(string)
				if !ok {
					log.Debug("no layout for timestamp parse, using default")
					layout = "2006-01-02 15:04:05"
				}

				model.Fields = append(model.Fields, &common.TimestampModelField{
					ModelField: modelField,
					Layout:     layout,
				})
			case "list":

				values, ok := f["values"].(map[interface{}]interface{})
				if !ok {
					log.Fatal("model field of type list must contain 'values' of string(key)=>int(value) pairs", zap.Any("values", values))
				}

				var defaultValue *int = nil
				def, ok := f["default"].(int)
				if ok {
					defaultValue = &def
				}

				listValues := make(map[string]int)
				for k := range values {
					key, ok := k.(string)
					if !ok {
						log.Fatal("cannot parse list key as string", zap.Any("key", k))
					}

					val, ok := values[k].(int)
					if !ok {
						log.Fatal("cannot parse list value as int", zap.String("key", key), zap.Any("value", values[k]))
					}
					listValues[key] = val
				}

				model.Fields = append(model.Fields, &common.ListModelField{
					ModelField: modelField,
					Values:     listValues,
					Default:    defaultValue,
				})

			case "ip2int":
				model.Fields = append(model.Fields, &common.IpToIntModelField{
					ModelField: modelField,
				})
			case "int16":
				model.Fields = append(model.Fields, &common.Int16ModelField{
					ModelField: modelField,
				})
			case "uint16":
				model.Fields = append(model.Fields, &common.UInt16ModelField{
					ModelField: modelField,
				})

			default:
				log.Fatal("unknown field type")
			}

		}
		s.ch.RegisterModel(r.Name, &model)
	}
}

func (s *syslogListener) ListenAndServe() error {

	s.msgChannel = make(syslog.LogPartsChannel)
	s.handler = syslog.NewChannelHandler(s.msgChannel)

	s.server = syslog.NewServer()
	s.server.SetFormat(syslog.Automatic)
	s.server.SetHandler(s.handler)
	if err := s.server.ListenUDP(s.address); err != nil {
		return err
	}

	if err := s.server.Boot(); err != nil {
		return err
	}

	go s.messageHandler(s.msgChannel)

	s.server.Wait()

	return nil
}

func (s *syslogListener) messageHandler(channel syslog.LogPartsChannel) {
	for logParts := range channel {
		content, ok := logParts["content"].(string)
		if !ok {
			s.log.Fatal("cannot parse content", zap.Any("log_parts", logParts))
		}
		for i := range s.rules {
			matches := s.rules[i].Regexp.FindAllStringSubmatch(content, -1)
			if len(matches) == 0 {
				continue
			}
			colsNum := len(s.rules[i].Fields)
			for m := range matches {
				if len(matches[m]) != colsNum+1 {
					s.log.Error("fields count mismatch in regexp and in fields definition",
						zap.Int("rule_fields_num", colsNum),
						zap.Int("regexp_results", len(matches[m])-1),
						zap.String("rule", s.rules[i].Name),
					)
					continue
				}

				msg := common.FlowMessage{
					Rule:   s.rules[i].Name,
					Fields: make(map[string]string, colsNum),
				}

				for j := range s.rules[i].Fields {

					fieldName, ok := s.rules[i].Fields[j]["name"].(string)
					if !ok {
						s.log.Fatal("field 'name' must be string", zap.Any("field", s.rules[i].Fields[j]))
					}

					msg.Fields[fieldName] = matches[m][j+1]
				}

				s.ch.Insert(&msg)

			}

		}
	}
}

func (s *syslogListener) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Kill()
	}

	return nil
}

func newSyslogService(p syslogParams) (syslogOutParams, error) {

	l := &syslogListener{
		timeout: p.Viper.GetDuration("syslog.timeout"),
		address: p.Viper.GetString("syslog.address"),
		log:     p.Logger,
		ch:      p.CH,

	}

	var svc service.Service
	if err := p.Viper.UnmarshalKey("syslog.rules", &l.rules, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			StringToRegexp(),
		))); err != nil {
		return syslogOutParams{}, err
	}
	l.registerModels()

	svc, err := web.NewListener(l,
		web.ListenerShutdownTimeout(p.Viper.GetDuration("syslog.shutdown_timeout")),
		web.ListenerName("syslog listener " + l.address),
	)
	return syslogOutParams{
		Service: svc,
	}, err
}

// StringToRegexp returns a DecodeHookFunc that converts
// strings to regexp.Regexp
func StringToRegexp() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf(regexp.Regexp{}) {
			return data, nil
		}

		// Convert it by parsing
		re, err := regexp.Compile(data.(string))
		return re, err
	}
}
