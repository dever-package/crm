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

// FrontFS 内嵌 CRM 前端插件资源。
//
// 本地开发时 front/dist 可能为空，dever run 会回退到 front/src/plugin.ts 启动 dev server；
// 发布前由 dever front build crm 写入 front/dist，运行时仍按 front/dist 前缀读取产物。
//
//go:embed front
var FrontFS embed.FS
