package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/plugins/backendplugin"
	"github.com/grafana/grafana/pkg/plugins/backendplugin/pluginextensionv2"
	"github.com/grafana/grafana/pkg/plugins/backendplugin/secretsmanagerplugin"
	"github.com/grafana/grafana/pkg/services/org"
)

type Plugin struct {
	JSONData

	Files     FS
	Class     Class

	// App fields
	IncludedInAppID string
	DefaultNavURL   string
	Pinned          bool

	// Signature fields
	Signature      SignatureStatus
	SignatureType  SignatureType
	SignatureOrg   string
	Parent         *Plugin
	Children       []*Plugin
	SignedFiles    PluginFiles
	SignatureError *SignatureError

	// SystemJS fields
	Module  string
	BaseURL string

	Renderer       pluginextensionv2.RendererPlugin
	SecretsManager secretsmanagerplugin.SecretsManagerPlugin
	client         backendplugin.Plugin
	log            log.Logger
}

type PluginDTO struct {
	JSONData

	files FS

	class Class

	// App fields
	IncludedInAppID string
	DefaultNavURL   string
	Pinned          bool

	// Signature fields
	Signature     SignatureStatus
	SignatureType SignatureType
	SignatureOrg  string
	signedFiles   PluginFiles

	// SystemJS fields
	Module  string
	BaseURL string

	supportsStreaming bool
}

func NewPluginDTO(jsonData JSONData, class Class) PluginDTO {
	return PluginDTO{
		class:    class,
		JSONData: jsonData,
	}
}

func (p PluginDTO) Upgradeable() bool {
	return !p.IsCorePlugin()
}

func (p PluginDTO) SupportsStreaming() bool {
	return p.supportsStreaming
}

func (p PluginDTO) IsApp() bool {
	return p.Type == App
}

func (p PluginDTO) IsCorePlugin() bool {
	return p.class == Core
}

func (p PluginDTO) IsExternalPlugin() bool {
	return p.class == External
}

func (p PluginDTO) IsSecretsManager() bool {
	return p.JSONData.Type == SecretsManager
}

func (p PluginDTO) Markdown(name string) []byte {
	path := mdFilepath(strings.ToUpper(name))
	exists := p.files.Exists(path)
	if !exists {
		path = mdFilepath(strings.ToLower(name))
	}

	exists = p.files.Exists(path)
	if !exists {
		return make([]byte, 0)
	}

	data, _ := p.files.Read(path)
	return data
}

func (p PluginDTO) File(name string) (io.ReadSeeker, time.Time, error) {
	f, err := p.files.Open(name)
	if err != nil {
		return nil, time.Time{}, err
	}

	fi, err := f.Stat()
	if err != nil {
		return nil, time.Time{}, err
	}

	b, err := io.ReadAll(f)
	if err != nil {
		return nil, time.Time{}, err
	}

	if err = f.Close(); err != nil {
		return nil, time.Time{}, err
	}

	return bytes.NewReader(b), fi.ModTime(), nil
}

func mdFilepath(mdFilename string) string {
	return filepath.Clean(filepath.Join("/", fmt.Sprintf("%s.md", mdFilename)))
}

// JSONData represents the plugin's plugin.json
type JSONData struct {
	// Common settings
	ID           string       `json:"id"`
	Type         Type         `json:"type"`
	Name         string       `json:"name"`
	Info         Info         `json:"info"`
	Dependencies Dependencies `json:"dependencies"`
	Includes     []*Includes  `json:"includes"`
	State        ReleaseState `json:"state,omitempty"`
	Category     string       `json:"category"`
	HideFromList bool         `json:"hideFromList,omitempty"`
	Preload      bool         `json:"preload"`
	Backend      bool         `json:"backend"`
	Routes       []*Route     `json:"routes"`

	// AccessControl settings
	Roles []RoleRegistration `json:"roles,omitempty"`

	// Panel settings
	SkipDataQuery bool `json:"skipDataQuery"`

	// App settings
	AutoEnabled bool `json:"autoEnabled"`

	// Datasource settings
	Annotations  bool            `json:"annotations"`
	Metrics      bool            `json:"metrics"`
	Alerting     bool            `json:"alerting"`
	Explore      bool            `json:"explore"`
	Table        bool            `json:"tables"`
	Logs         bool            `json:"logs"`
	Tracing      bool            `json:"tracing"`
	QueryOptions map[string]bool `json:"queryOptions,omitempty"`
	BuiltIn      bool            `json:"builtIn,omitempty"`
	Mixed        bool            `json:"mixed,omitempty"`
	Streaming    bool            `json:"streaming"`
	SDK          bool            `json:"sdk,omitempty"`

	// Backend (Datasource + Renderer + SecretsManager)
	Executable string `json:"executable,omitempty"`
}

func (d JSONData) DashboardIncludes() []*Includes {
	result := []*Includes{}
	for _, include := range d.Includes {
		if include.Type == TypeDashboard {
			result = append(result, include)
		}
	}

	return result
}

// Route describes a plugin route that is defined in
// the plugin.json file for a plugin.
type Route struct {
	Path         string          `json:"path"`
	Method       string          `json:"method"`
	ReqRole      org.RoleType    `json:"reqRole"`
	URL          string          `json:"url"`
	URLParams    []URLParam      `json:"urlParams"`
	Headers      []Header        `json:"headers"`
	AuthType     string          `json:"authType"`
	TokenAuth    *JWTTokenAuth   `json:"tokenAuth"`
	JwtTokenAuth *JWTTokenAuth   `json:"jwtTokenAuth"`
	Body         json.RawMessage `json:"body"`
}

// Header describes an HTTP header that is forwarded with
// the proxied request for a plugin route
type Header struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// URLParam describes query string parameters for
// a url in a plugin route
type URLParam struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// JWTTokenAuth struct is both for normal Token Auth and JWT Token Auth with
// an uploaded JWT file.
type JWTTokenAuth struct {
	Url    string            `json:"url"`
	Scopes []string          `json:"scopes"`
	Params map[string]string `json:"params"`
}

func (p *Plugin) PluginID() string {
	return p.ID
}

func (p *Plugin) Logger() log.Logger {
	return p.log
}

func (p *Plugin) SetLogger(l log.Logger) {
	p.log = l
}

func (p *Plugin) Start(ctx context.Context) error {
	if p.client == nil {
		return fmt.Errorf("could not start plugin %s as no plugin client exists", p.ID)
	}
	return p.client.Start(ctx)
}

func (p *Plugin) Stop(ctx context.Context) error {
	if p.client == nil {
		return nil
	}
	return p.client.Stop(ctx)
}

func (p *Plugin) IsManaged() bool {
	if p.client != nil {
		return p.client.IsManaged()
	}
	return false
}

func (p *Plugin) Decommission() error {
	if p.client != nil {
		return p.client.Decommission()
	}
	return nil
}

func (p *Plugin) IsDecommissioned() bool {
	if p.client != nil {
		return p.client.IsDecommissioned()
	}
	return false
}

func (p *Plugin) Exited() bool {
	if p.client != nil {
		return p.client.Exited()
	}
	return false
}

func (p *Plugin) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	pluginClient, ok := p.Client()
	if !ok {
		return nil, backendplugin.ErrPluginUnavailable
	}
	return pluginClient.QueryData(ctx, req)
}

func (p *Plugin) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	pluginClient, ok := p.Client()
	if !ok {
		return backendplugin.ErrPluginUnavailable
	}
	return pluginClient.CallResource(ctx, req, sender)
}

func (p *Plugin) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	pluginClient, ok := p.Client()
	if !ok {
		return nil, backendplugin.ErrPluginUnavailable
	}
	return pluginClient.CheckHealth(ctx, req)
}

func (p *Plugin) CollectMetrics(ctx context.Context, req *backend.CollectMetricsRequest) (*backend.CollectMetricsResult, error) {
	pluginClient, ok := p.Client()
	if !ok {
		return nil, backendplugin.ErrPluginUnavailable
	}
	return pluginClient.CollectMetrics(ctx, req)
}

func (p *Plugin) SubscribeStream(ctx context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	pluginClient, ok := p.Client()
	if !ok {
		return nil, backendplugin.ErrPluginUnavailable
	}
	return pluginClient.SubscribeStream(ctx, req)
}

func (p *Plugin) PublishStream(ctx context.Context, req *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
	pluginClient, ok := p.Client()
	if !ok {
		return nil, backendplugin.ErrPluginUnavailable
	}
	return pluginClient.PublishStream(ctx, req)
}

func (p *Plugin) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	pluginClient, ok := p.Client()
	if !ok {
		return backendplugin.ErrPluginUnavailable
	}
	return pluginClient.RunStream(ctx, req, sender)
}

func (p *Plugin) RegisterClient(c backendplugin.Plugin) {
	p.client = c
}

func (p *Plugin) Client() (PluginClient, bool) {
	if p.client != nil {
		return p.client, true
	}
	return nil, false
}

func (p *Plugin) ExecutablePath() string {
	if p.IsRenderer() {
		return pluginExecutable(p, "plugin_start")
	}

	if p.IsSecretsManager() {
		return pluginExecutable(p, "secrets_plugin_start")
	}

	return pluginExecutable(p, p.Executable)
}

func pluginExecutable(p *Plugin, f string) string {
	os := strings.ToLower(runtime.GOOS)
	arch := runtime.GOARCH
	extension := ""

	if os == "windows" {
		extension = ".exe"
	}
	fp, exists := p.Files.FullPath(fmt.Sprintf("%s_%s_%s%s", f, os, strings.ToLower(arch), extension))
	if !exists {
		return ""
	}
	return fp
}

type PluginClient interface {
	backend.QueryDataHandler
	backend.CollectMetricsHandler
	backend.CheckHealthHandler
	backend.CallResourceHandler
	backend.StreamHandler
}

func (p *Plugin) ToDTO() PluginDTO {
	return PluginDTO{
		files:             p.Files,
		class:             p.Class,
		signedFiles:       p.SignedFiles,
		supportsStreaming: p.client != nil && p.client.(backend.StreamHandler) != nil,
		JSONData:          p.JSONData,
		IncludedInAppID:   p.IncludedInAppID,
		DefaultNavURL:     p.DefaultNavURL,
		Pinned:            p.Pinned,
		Signature:         p.Signature,
		SignatureType:     p.SignatureType,
		SignatureOrg:      p.SignatureOrg,
		Module:            p.Module,
		BaseURL:           p.BaseURL,
	}
}

func (p *Plugin) StaticRoute() *StaticRoute {
	if p.IsCorePlugin() {
		return nil
	}

	return &StaticRoute{Directory: p.Files.Base(), PluginID: p.ID}
}

func (p *Plugin) IsRenderer() bool {
	return p.Type == "renderer"
}

func (p *Plugin) IsSecretsManager() bool {
	return p.Type == "secretsmanager"
}

func (p *Plugin) IsDataSource() bool {
	return p.Type == "datasource"
}

func (p *Plugin) IsPanel() bool {
	return p.Type == "panel"
}

func (p *Plugin) IsApp() bool {
	return p.Type == "app"
}

func (p *Plugin) IsCorePlugin() bool {
	return p.Class == Core
}

func (p *Plugin) IsBundledPlugin() bool {
	return p.Class == Bundled
}

func (p *Plugin) IsExternalPlugin() bool {
	return p.Class == External
}

func (p *Plugin) Manifest() []byte {
	f, err := p.Files.Open("MANIFEST.txt")
	if err != nil {
		return []byte{}
	}

	m, err := io.ReadAll(f)
	if err != nil {
		return []byte{}
	}
	return m
}

type Class string

const (
	Core     Class = "core"
	Bundled  Class = "bundled"
	External Class = "external"
	Remote   Class = "remote"
)

var PluginTypes = []Type{
	DataSource,
	Panel,
	App,
	Renderer,
	SecretsManager,
}

type Type string

const (
	DataSource     Type = "datasource"
	Panel          Type = "panel"
	App            Type = "app"
	Renderer       Type = "renderer"
	SecretsManager Type = "secretsmanager"
)

func (pt Type) IsValid() bool {
	switch pt {
	case DataSource, Panel, App, Renderer, SecretsManager:
		return true
	}
	return false
}
