package route

import (
	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/go-openapi/spec"
)

func AddSwagger(container *restful.Container) {
	cfg := restfulspec.Config{
		WebServices:                   container.RegisteredWebServices(),
		APIPath:                       "/bot/swagger.json",
		PostBuildSwaggerObjectHandler: enrichSwaggerObject,
	}
	container.Add(restfulspec.NewOpenAPIService(cfg))
}

func enrichSwaggerObject(swo *spec.Swagger) {
	swo.Info = &spec.Info{
		InfoProps: spec.InfoProps{
			Title:       "Chatbot REST API",
			Description: "Resource for Chatbot",
			Contact: &spec.ContactInfo{
				ContactInfoProps: spec.ContactInfoProps{
					URL: "https://github.com/sysatom/flowbot",
				},
			},
			License: &spec.License{
				LicenseProps: spec.LicenseProps{
					Name: "MIT",
					URL:  "https://github.com/sysatom/flowbot/blob/main/LICENSE",
				},
			},
			Version: "1.0.0",
		},
	}
	swo.Tags = []spec.Tag{{TagProps: spec.TagProps{
		Name:        "Chatbot",
		Description: "Chatbot"}}}
	swo.SecurityDefinitions = map[string]*spec.SecurityScheme{
		"BearerToken": {
			SecuritySchemeProps: spec.SecuritySchemeProps{
				Type:        "apiKey",
				Name:        "authorization",
				In:          "header",
				Description: "Bearer Token authentication",
			},
		},
	}
	swo.Security = []map[string][]string{
		{
			"BearerToken": {},
		},
	}
}
