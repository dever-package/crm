package crm

import "embed"

// ManifestFS 内嵌 CRM 组件声明。
//
//go:embed dever.json
var ManifestFS embed.FS

// PageFS 内嵌 CRM 后台页面配置。
//
//go:embed front/page/*/*/*.json
var PageFS embed.FS

// FrontFS 内嵌 CRM 前端插件静态产物。
//
//go:embed front/dist
var FrontFS embed.FS
