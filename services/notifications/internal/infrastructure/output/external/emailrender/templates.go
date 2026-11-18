package emailrender

const processedHTML = `<h1>{{.Greeting}} 👋</h1>
<p>¡Soy Ori! Te cuento que {{.Photo}} del lote <strong>{{.LotCode}}</strong> (etapa {{.Stage}}) ya fue procesada y quedó registrada en su trazabilidad.</p>
<p><a href="{{.URL}}" style="display:inline-block; padding:12px 24px; background-color:#6B4F3A; color:#ffffff; text-decoration:none; border-radius:6px; font-weight:bold;">Ver el lote</a></p>
<p>— Ori</p>`

const processedText = `{{.Greeting}}

Soy Ori. Te cuento que {{.Photo}} del lote {{.LotCode}} (etapa {{.Stage}}) ya fue procesada y quedó registrada en su trazabilidad.

Ver el lote: {{.URL}}

— Ori`

const rejectedHTML = `<h1>{{.Greeting}} 👋</h1>
<p>¡Soy Ori! No pudimos procesar {{.Photo}} del lote <strong>{{.LotCode}}</strong> (etapa {{.Stage}}).</p>
{{if .Reason}}<p><strong>Motivo:</strong> {{.Reason}}</p>{{end}}
<p>Puedes subir otra foto desde el detalle del lote.</p>
<p><a href="{{.URL}}" style="display:inline-block; padding:12px 24px; background-color:#6B4F3A; color:#ffffff; text-decoration:none; border-radius:6px; font-weight:bold;">Ir al lote</a></p>
<p>— Ori</p>`

const rejectedText = `{{.Greeting}}

Soy Ori. No pudimos procesar {{.Photo}} del lote {{.LotCode}} (etapa {{.Stage}}).
{{if .Reason}}
Motivo: {{.Reason}}
{{end}}
Puedes subir otra foto desde el detalle del lote.

Ir al lote: {{.URL}}

— Ori`
