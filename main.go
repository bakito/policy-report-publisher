package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	kspAPI "github.com/kubearmor/KubeArmor/pkg/KubeArmorController/api/security.kubearmor.com/v1"
	policyreportv1alpha2 "github.com/kyverno/kyverno/api/policyreport/v1alpha2"
	"github.com/kyverno/policy-reporter-plugins/example/api"
	_ "github.com/kyverno/policy-reporter-plugins/example/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"k8s.io/client-go/kubernetes/scheme"
)

//	@title			Policy Reporter UI Plugin Example API
//	@version		0.0.1
//	@description	This is an example API for the Policy Reporter UI Plugin Interface.

//	@contact.name	Frank Jogeleit
//	@contact.email	frank.jogeleit@web.de

// @host		localhost:8080
// @BasePath	/api
func main() {

	_ = policyreportv1alpha2.AddToScheme(scheme.Scheme)
	_ = kspAPI.AddToScheme(scheme.Scheme)

	client := api.NewClient()

	r := gin.Default()

	g := r.Group("api/v1")

	g.GET("policies", func(ctx *gin.Context) {
		policies, err := client.GetPolicies(ctx)
		if err != nil {
			_ = ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		ctx.JSON(http.StatusOK, policies)
	})

	g.GET("policies/*policy", func(ctx *gin.Context) {
		policy, err := client.GetPolicy(ctx, strings.Trim(ctx.Param("policy"), "/"))
		if err != nil {
			_ = ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		ctx.JSON(http.StatusOK, policy)
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.Run(":8080")
}
