package crm

import "embed"

// ManifestFS 内嵌 CRM 组件声明。
//
//go:embed dever.json
var ManifestFS embed.FS

// PageFS 内嵌 CRM 页面配置和站点默认静态资源。
//
//go:embed front/page/*/*.json front/page/*/*/*.json front/assets
var PageFS embed.FS

// FrontFS 内嵌 CRM 前端插件静态产物。
//
//go:embed front/dist
var FrontFS embed.FS
