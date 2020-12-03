package clickhouse

import "github.com/im-kulikov/helium/module"

// Module application
var Module = module.Module{
	{Constructor: newSettings},
	{Constructor: newService},
}
