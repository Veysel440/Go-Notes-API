package openapi

import (
	"embed"
	"fmt"
	"net/http"
)

//go:embed openapi.yaml
var specFS embed.FS

func Spec() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := specFS.ReadFile("../../openapi/openapi.yaml")
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		w.Write(b)
	})
}

func UI() http.Handler {
	const tpl = `<!doctype html><html><head>
<meta charset="utf-8"/>
<title>OpenAPI</title>
<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist/swagger-ui.css">
</head><body>
<div id="swagger"></div>
<script src="https://unpkg.com/swagger-ui-dist/swagger-ui-bundle.js"></script>
<script>
window.ui = SwaggerUIBundle({ url: "/openapi.yaml", dom_id: "#swagger" });
</script></body></html>`
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, tpl)
	})
}
