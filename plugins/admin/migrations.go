package admin

import "github.com/theinventorylib/aegis/plugins"

// GetMigrations returns plugin migrations
func (p *Plugin) GetMigrations() []plugins.Migration {
	return []plugins.Migration{}
}
