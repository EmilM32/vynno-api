package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/gin-gonic/gin"
)

// op is OpenAPI metadata recorded at the same call that mounts a Gin handler.
type op struct {
	Summary     string
	Description string
	Tags        []string
	Public      bool
	Body        any
	Success     any
	SuccessCode int
	SuccessDesc string
	Empty       bool
	Binary      string
	Errors      []string
	Query       []queryParam
	Multipart   string
	SetCookie   bool
	ClearCookie bool
}

type queryParam struct {
	Name        string
	Description string
	Type        string
}

type documentedOp struct {
	method string
	path   string
	op     op
}

type ginRouter interface {
	Handle(httpMethod, relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
	BasePath() string
}

func (s *Server) route(g ginRouter, method, path string, h gin.HandlerFunc, o op) {
	g.Handle(method, path, h)
	full := strings.TrimSuffix(g.BasePath(), "/") + path
	if !strings.HasPrefix(full, "/") {
		full = "/" + full
	}
	s.ops = append(s.ops, documentedOp{method: method, path: full, op: o})
}

func (s *Server) buildSpec() {
	raw, err := marshalOpenAPI(s.ops)
	if err != nil {
		panic("openapi: " + err.Error())
	}
	s.specJSON = raw
}

func (s *Server) serveOpenAPI(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "application/json; charset=utf-8", s.specJSON)
}

func marshalOpenAPI(ops []documentedOp) ([]byte, error) {
	book := newSchemaBook()
	paths := map[string]map[string]oaOperation{}
	for _, d := range ops {
		p := openAPIPath(d.path)
		if paths[p] == nil {
			paths[p] = map[string]oaOperation{}
		}
		paths[p][strings.ToLower(d.method)] = buildOperation(d, book)
	}

	doc := oaDoc{
		OpenAPI: "3.0.3",
		Info: oaInfo{
			Title:   "Vynno API",
			Version: "1.0.0",
			Description: strings.TrimSpace(`
HTTP JSON API for Vynno (VIN-oh). The SPA contract is docs/api-contract.md; this document is generated from the same Go route registrations that mount the handlers.

**Try it out.** Open this UI at PUBLIC_API_ORIGIN (production: http://localhost:8080/swagger/, playground: http://127.0.0.1:8081/swagger/). Login via POST /v1/auth/login — the HttpOnly cookie vynno_session is stored by the browser and sent on later requests. localhost and 127.0.0.1 are different origins; cookie mutations from the wrong host return 401.

**Authorize → Bearer** is the curl/tests alternative (ADR-0008). The SPA uses the cookie only.

**Rules the server enforces**
- At most one session with status active or paused.
- Session and project lifecycle use verb routes (/pause, /resume, /stop, /archive, /restore).
- Cannot archive or delete the last active project.
- Hard-delete a project or activity type only when it has zero sessions.
- IDs are opaque UUID strings.
- Absent optionals on responses are JSON null. PATCH bodies: omit a field to leave it unchanged; code: null clears a project code.
`),
		},
		Servers: []oaServer{{URL: "/", Description: "This process"}},
		Tags: []oaTag{
			{Name: "Ops", Description: "Process health; not part of the SPA /v1 contract."},
			{Name: "Auth", Description: "Register, login, logout. Cookie vynno_session."},
			{Name: "Profile", Description: "Current user profile and avatar."},
			{Name: "Projects", Description: "Projects the signed-in user owns."},
			{Name: "Activity types", Description: "Per-user activity type dictionary."},
			{Name: "Sessions", Description: "Focus sessions (timer)."},
		},
		Paths: paths,
		Components: oaComponents{
			Schemas: book.schemas,
			SecuritySchemes: map[string]oaSecurityScheme{
				"cookieAuth": {
					Type:        "apiKey",
					In:          "cookie",
					Name:        sessionCookie,
					Description: "HttpOnly session cookie set by login/register.",
				},
				"bearerAuth": {
					Type:         "http",
					Scheme:       "bearer",
					Description:  "Same opaque token as the cookie; for curl and tests.",
					BearerFormat: "opaque",
				},
			},
		},
		Security: []map[string][]string{
			{"cookieAuth": {}},
			{"bearerAuth": {}},
		},
	}
	book.ensureErrorEnvelope()
	return json.MarshalIndent(doc, "", "  ")
}

func buildOperation(d documentedOp, book *schemaBook) oaOperation {
	o := d.op
	successCode := o.SuccessCode
	if successCode == 0 {
		if o.Empty {
			successCode = http.StatusNoContent
		} else {
			successCode = http.StatusOK
		}
	}
	successDesc := o.SuccessDesc
	if successDesc == "" {
		successDesc = http.StatusText(successCode)
	}

	out := oaOperation{
		Tags:        o.Tags,
		Summary:     o.Summary,
		Description: o.Description,
		OperationID: operationID(d.method, d.path),
		Responses:   map[string]oaResponse{},
	}
	if o.Public {
		empty := []map[string][]string{}
		out.Security = &empty
	}

	if strings.Contains(d.path, ":") {
		for _, name := range pathParamNames(d.path) {
			out.Parameters = append(out.Parameters, oaParam{
				Name:        name,
				In:          "path",
				Required:    true,
				Description: "Opaque UUID.",
				Schema:      oaSchema{Type: "string", Format: "uuid"},
			})
		}
	}
	for _, q := range o.Query {
		out.Parameters = append(out.Parameters, oaParam{
			Name:        q.Name,
			In:          "query",
			Required:    false,
			Description: q.Description,
			Schema:      oaSchema{Type: q.Type},
		})
	}

	if o.Multipart != "" {
		out.RequestBody = &oaRequestBody{
			Required: true,
			Content: map[string]oaMedia{
				"multipart/form-data": {Schema: oaSchema{
					Type: "object",
					Properties: map[string]oaSchema{
						o.Multipart: {Type: "string", Format: "binary", Description: "jpeg, png, or webp; max 1 MiB."},
					},
					Required: []string{o.Multipart},
				}},
			},
		}
	} else if o.Body != nil {
		out.RequestBody = &oaRequestBody{
			Required: true,
			Content: map[string]oaMedia{
				"application/json": {Schema: book.refFor(o.Body)},
			},
		}
	}

	success := oaResponse{Description: successDesc}
	switch {
	case o.Empty:
		// no content
	case o.Binary != "":
		success.Content = map[string]oaMedia{
			o.Binary: {Schema: oaSchema{Type: "string", Format: "binary"}},
		}
	case o.Success != nil:
		success.Content = map[string]oaMedia{
			"application/json": {Schema: book.refFor(o.Success)},
		}
	}
	if o.SetCookie {
		success.Headers = map[string]oaHeader{
			"Set-Cookie": {
				Description: "HttpOnly vynno_session. rememberMe true (default) sets Max-Age 30 days; false is a session cookie.",
				Schema:      oaSchema{Type: "string", Example: "vynno_session=…; Path=/; HttpOnly; SameSite=Lax"},
			},
		}
	}
	if o.ClearCookie {
		success.Headers = map[string]oaHeader{
			"Set-Cookie": {
				Description: "Clears vynno_session.",
				Schema:      oaSchema{Type: "string"},
			},
		}
	}
	out.Responses[strconv.Itoa(successCode)] = success

	codes := append([]string(nil), o.Errors...)
	if !o.Public {
		codes = append(codes, domain.CodeUnauthorized)
	}
	if o.Body != nil {
		codes = append(codes, domain.CodeInvalidJSON, domain.CodeInvalidBody)
	}
	codes = uniqueStrings(codes)
	byStatus := map[int][]string{}
	for _, code := range codes {
		st := statusFor(code)
		byStatus[st] = append(byStatus[st], code)
	}
	statuses := make([]int, 0, len(byStatus))
	for st := range byStatus {
		statuses = append(statuses, st)
	}
	sort.Ints(statuses)
	for _, st := range statuses {
		list := byStatus[st]
		sort.Strings(list)
		out.Responses[strconv.Itoa(st)] = oaResponse{
			Description: "Error codes: " + strings.Join(list, ", ") + ".",
			Content: map[string]oaMedia{
				"application/json": {Schema: oaSchema{Ref: "#/components/schemas/ErrorEnvelope"}},
			},
		}
	}
	return out
}

func openAPIPath(ginPath string) string {
	parts := strings.Split(ginPath, "/")
	for i, p := range parts {
		if strings.HasPrefix(p, ":") {
			parts[i] = "{" + p[1:] + "}"
		}
	}
	return strings.Join(parts, "/")
}

func pathParamNames(ginPath string) []string {
	var names []string
	for _, p := range strings.Split(ginPath, "/") {
		if strings.HasPrefix(p, ":") {
			names = append(names, p[1:])
		}
	}
	return names
}

func operationID(method, ginPath string) string {
	p := strings.Trim(openAPIPath(ginPath), "/")
	p = strings.ReplaceAll(p, "/", "_")
	p = strings.ReplaceAll(p, "{", "")
	p = strings.ReplaceAll(p, "}", "")
	return strings.ToLower(method) + "_" + p
}

func uniqueStrings(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

type oaDoc struct {
	OpenAPI    string                            `json:"openapi"`
	Info       oaInfo                            `json:"info"`
	Servers    []oaServer                        `json:"servers"`
	Tags       []oaTag                           `json:"tags"`
	Paths      map[string]map[string]oaOperation `json:"paths"`
	Components oaComponents                      `json:"components"`
	Security   []map[string][]string             `json:"security"`
}

type oaInfo struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

type oaServer struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

type oaTag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type oaOperation struct {
	Tags        []string               `json:"tags,omitempty"`
	Summary     string                 `json:"summary"`
	Description string                 `json:"description,omitempty"`
	OperationID string                 `json:"operationId"`
	Security    *[]map[string][]string `json:"security,omitempty"`
	Parameters  []oaParam              `json:"parameters,omitempty"`
	RequestBody *oaRequestBody         `json:"requestBody,omitempty"`
	Responses   map[string]oaResponse  `json:"responses"`
}

type oaParam struct {
	Name        string   `json:"name"`
	In          string   `json:"in"`
	Required    bool     `json:"required"`
	Description string   `json:"description,omitempty"`
	Schema      oaSchema `json:"schema"`
}

type oaRequestBody struct {
	Required bool               `json:"required"`
	Content  map[string]oaMedia `json:"content"`
}

type oaResponse struct {
	Description string              `json:"description"`
	Headers     map[string]oaHeader `json:"headers,omitempty"`
	Content     map[string]oaMedia  `json:"content,omitempty"`
}

type oaHeader struct {
	Description string   `json:"description,omitempty"`
	Schema      oaSchema `json:"schema"`
}

type oaMedia struct {
	Schema oaSchema `json:"schema"`
}

type oaComponents struct {
	Schemas         map[string]oaSchema         `json:"schemas"`
	SecuritySchemes map[string]oaSecurityScheme `json:"securitySchemes"`
}

type oaSecurityScheme struct {
	Type         string `json:"type"`
	In           string `json:"in,omitempty"`
	Name         string `json:"name,omitempty"`
	Scheme       string `json:"scheme,omitempty"`
	BearerFormat string `json:"bearerFormat,omitempty"`
	Description  string `json:"description,omitempty"`
}

type oaSchema struct {
	Type                 string              `json:"type,omitempty"`
	Format               string              `json:"format,omitempty"`
	Description          string              `json:"description,omitempty"`
	Properties           map[string]oaSchema `json:"properties,omitempty"`
	Required             []string            `json:"required,omitempty"`
	Items                *oaSchema           `json:"items,omitempty"`
	Ref                  string              `json:"$ref,omitempty"`
	Nullable             bool                `json:"nullable,omitempty"`
	Enum                 []string            `json:"enum,omitempty"`
	Example              any                 `json:"example,omitempty"`
	AdditionalProperties any                 `json:"additionalProperties,omitempty"`
	AllOf                []oaSchema          `json:"allOf,omitempty"`
}

type schemaBook struct {
	schemas map[string]oaSchema
}

func newSchemaBook() *schemaBook {
	return &schemaBook{schemas: map[string]oaSchema{}}
}

func (b *schemaBook) ensureErrorEnvelope() {
	_ = b.refFor(errorEnvelope{})
}

func (b *schemaBook) refFor(v any) oaSchema {
	if v == nil {
		return oaSchema{}
	}
	return b.refForType(reflect.TypeOf(v), false)
}

func (b *schemaBook) refForType(t reflect.Type, nullable bool) oaSchema {
	for t.Kind() == reflect.Pointer {
		nullable = true
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		s := b.scalarSchema(t)
		s.Nullable = nullable
		return s
	}
	name := prettySchemaName(t)
	if _, ok := b.schemas[name]; !ok {
		b.schemas[name] = oaSchema{Type: "object"}
		b.schemas[name] = b.structSchema(t)
	}
	ref := oaSchema{Ref: "#/components/schemas/" + name}
	if !nullable {
		return ref
	}
	return oaSchema{Nullable: true, AllOf: []oaSchema{ref}}
}

func (b *schemaBook) structSchema(t reflect.Type) oaSchema {
	props := map[string]oaSchema{}
	var required []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" && !f.Anonymous {
			continue
		}
		tag := f.Tag.Get("json")
		if tag == "-" {
			continue
		}
		name, omitempty := jsonField(tag, f.Name)
		if name == "" {
			continue
		}
		ft := f.Type
		nullable := ft.Kind() == reflect.Pointer
		s := b.fieldSchema(ft)
		applyFieldHints(t, name, &s)
		props[name] = s
		if !omitempty && !nullable {
			required = append(required, name)
		}
	}
	sort.Strings(required)
	return oaSchema{Type: "object", Properties: props, Required: required}
}

func (b *schemaBook) fieldSchema(t reflect.Type) oaSchema {
	nullable := false
	for t.Kind() == reflect.Pointer {
		nullable = true
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Struct:
		return b.refForType(t, nullable)
	case reflect.Slice, reflect.Array:
		elem := b.fieldSchema(t.Elem())
		s := oaSchema{Type: "array", Items: &elem}
		s.Nullable = nullable
		return s
	default:
		s := b.scalarSchema(t)
		s.Nullable = nullable
		return s
	}
}

func (b *schemaBook) scalarSchema(t reflect.Type) oaSchema {
	switch t.Kind() {
	case reflect.String:
		return oaSchema{Type: "string"}
	case reflect.Bool:
		return oaSchema{Type: "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return oaSchema{Type: "integer"}
	case reflect.Float32, reflect.Float64:
		return oaSchema{Type: "number"}
	case reflect.Map:
		return oaSchema{Type: "object", AdditionalProperties: true}
	default:
		return oaSchema{Type: "object"}
	}
}

func jsonField(tag, fallback string) (name string, omitempty bool) {
	if tag == "" {
		return unexportedJSONName(fallback), false
	}
	parts := strings.Split(tag, ",")
	name = parts[0]
	if name == "" {
		name = unexportedJSONName(fallback)
	}
	for _, p := range parts[1:] {
		if p == "omitempty" {
			omitempty = true
		}
	}
	return name, omitempty
}

func unexportedJSONName(field string) string {
	if field == "" {
		return field
	}
	return strings.ToLower(field[:1]) + field[1:]
}

func prettySchemaName(t reflect.Type) string {
	name := t.Name()
	if strings.HasPrefix(name, "listDTO[") {
		if f, ok := t.FieldByName("Items"); ok {
			elem := f.Type
			for elem.Kind() == reflect.Slice || elem.Kind() == reflect.Pointer {
				elem = elem.Elem()
			}
			return prettySchemaName(elem) + "List"
		}
	}
	name = strings.TrimSuffix(name, "DTO")
	switch {
	case strings.HasSuffix(name, "Body"):
		name = strings.TrimSuffix(name, "Body") + "Request"
	case name == "errorEnvelope":
		return "ErrorEnvelope"
	case name == "errorBody":
		return "ErrorBody"
	}
	if name == "" {
		return "Object"
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

func applyFieldHints(parent reflect.Type, jsonName string, s *oaSchema) {
	switch prettySchemaName(parent) {
	case "Session":
		if jsonName == "status" {
			s.Enum = []string{"active", "paused", "stopped"}
		}
	case "ActivityType", "CreateActivityTypeRequest", "UpdateActivityTypeRequest":
		if jsonName == "color" {
			s.Enum = append([]string(nil), domain.ActivityColorTokens...)
		}
	}
}

func specPathsFromJSON(raw []byte) (map[string]map[string]struct{}, error) {
	var doc struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	out := map[string]map[string]struct{}{}
	for p, methods := range doc.Paths {
		out[p] = map[string]struct{}{}
		for m := range methods {
			if m == "parameters" {
				continue
			}
			out[p][strings.ToUpper(m)] = struct{}{}
		}
	}
	return out, nil
}

func formatPathSet(m map[string]map[string]struct{}) string {
	var keys []string
	for p, methods := range m {
		var ms []string
		for method := range methods {
			ms = append(ms, method)
		}
		sort.Strings(ms)
		keys = append(keys, fmt.Sprintf("%s %s", strings.Join(ms, ","), p))
	}
	sort.Strings(keys)
	return strings.Join(keys, "\n")
}
