package emailrender

import (
	"bytes"
	htmltemplate "html/template"
	"strings"
	texttemplate "text/template"

	"rutadeorigen/notifications/internal/application/output/ports"
	"rutadeorigen/notifications/internal/domain/notification"
	"rutadeorigen/notifications/internal/platform/exception"
)

var _ ports.EmailRenderer = (*Renderer)(nil)

const lotPath = "/lotes/"

var stageLabels = map[string]string{
	"farm":      "Finca",
	"roasting":  "Tostión",
	"packaging": "Empaque",
}

type view struct {
	Greeting string
	Photo    string
	LotCode  string
	Stage    string
	Reason   string
	URL      string
}

type compiled struct {
	subject func(lotCode string) string
	html    *htmltemplate.Template
	text    *texttemplate.Template
}

type Renderer struct {
	baseURL  string
	compiled map[notification.Kind]compiled
}

func New(publicBaseURL string) (*Renderer, error) {
	definitions := map[notification.Kind]struct {
		subject func(string) string
		html    string
		text    string
	}{
		notification.KindEvidenceProcessed: {
			subject: func(code string) string { return "Tu foto del lote " + code + " ya está publicada" },
			html:    processedHTML,
			text:    processedText,
		},
		notification.KindEvidenceRejected: {
			subject: func(code string) string { return "No pudimos procesar tu foto del lote " + code },
			html:    rejectedHTML,
			text:    rejectedText,
		},
	}

	result := make(map[notification.Kind]compiled, len(definitions))
	for kind, definition := range definitions {
		htmlTemplate, err := htmltemplate.New(kind.String() + ".html").Parse(definition.html)
		if err != nil {
			return nil, exception.New(exception.CodeEmailRenderFailed).WithTechnical("parse html template").WithCause(err)
		}
		textTemplate, err := texttemplate.New(kind.String() + ".txt").Parse(definition.text)
		if err != nil {
			return nil, exception.New(exception.CodeEmailRenderFailed).WithTechnical("parse text template").WithCause(err)
		}
		result[kind] = compiled{subject: definition.subject, html: htmlTemplate, text: textTemplate}
	}

	return &Renderer{baseURL: strings.TrimRight(strings.TrimSpace(publicBaseURL), "/"), compiled: result}, nil
}

func (r *Renderer) Render(kind notification.Kind, content ports.EmailContent) (ports.RenderedEmail, error) {
	templates, found := r.compiled[kind]
	if !found {
		return ports.RenderedEmail{}, exception.New(exception.CodeEmailRenderFailed).WithTechnical("unknown notification kind")
	}

	data := r.view(content)

	var htmlBody, textBody bytes.Buffer
	if err := templates.html.Execute(&htmlBody, data); err != nil {
		return ports.RenderedEmail{}, exception.New(exception.CodeEmailRenderFailed).WithTechnical("render html").WithCause(err)
	}
	if err := templates.text.Execute(&textBody, data); err != nil {
		return ports.RenderedEmail{}, exception.New(exception.CodeEmailRenderFailed).WithTechnical("render text").WithCause(err)
	}

	return ports.RenderedEmail{
		Subject: templates.subject(data.LotCode),
		Text:    textBody.String(),
		HTML:    htmlBody.String(),
	}, nil
}

func (r *Renderer) view(content ports.EmailContent) view {
	greeting := "¡Hola!"
	if name := strings.TrimSpace(content.RecipientName); name != "" {
		greeting = "¡Hola, " + name + "!"
	}

	photo := "tu foto"
	if fileName := strings.TrimSpace(content.FileName); fileName != "" {
		photo = "tu foto «" + fileName + "»"
	}

	stage := content.Stage
	if label, found := stageLabels[strings.ToLower(strings.TrimSpace(content.Stage))]; found {
		stage = label
	}

	return view{
		Greeting: greeting,
		Photo:    photo,
		LotCode:  content.LotCode,
		Stage:    stage,
		Reason:   strings.TrimSpace(content.Reason),
		URL:      r.baseURL + lotPath + content.LotID.String(),
	}
}
