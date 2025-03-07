/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/apache/incubator-devlake/api/blueprints"
	"github.com/apache/incubator-devlake/api/domainlayer"
	"github.com/apache/incubator-devlake/api/ping"
	"github.com/apache/incubator-devlake/api/pipelines"
	"github.com/apache/incubator-devlake/api/plugininfo"
	"github.com/apache/incubator-devlake/api/push"
	"github.com/apache/incubator-devlake/api/shared"
	"github.com/apache/incubator-devlake/api/task"
	"github.com/apache/incubator-devlake/api/version"
	"github.com/apache/incubator-devlake/plugins/core"
	"github.com/apache/incubator-devlake/services"
	"github.com/gin-gonic/gin"
)

func RegisterRouter(r *gin.Engine) {
	r.GET("/pipelines", pipelines.Index)
	r.POST("/pipelines", pipelines.Post)
	r.GET("/pipelines/:pipelineId", pipelines.Get)
	r.PATCH("/blueprints/:blueprintId", blueprints.Patch)
	r.POST("/blueprints/:blueprintId/trigger", blueprints.Trigger)
	// r.DELETE("/blueprints/:blueprintId", blueprints.Delete)

	r.GET("/blueprints", blueprints.Index)
	r.POST("/blueprints", blueprints.Post)
	r.GET("/blueprints/:blueprintId", blueprints.Get)
	r.GET("/blueprints/:blueprintId/pipelines", blueprints.GetBlueprintPipelines)
	r.DELETE("/pipelines/:pipelineId", pipelines.Delete)
	r.GET("/pipelines/:pipelineId/tasks", task.Index)

	r.GET("/pipelines/:pipelineId/logging.tar.gz", pipelines.DownloadLogs)

	r.GET("/ping", ping.Get)
	r.GET("/version", version.Get)
	r.POST("/push/:tableName", push.Post)
	r.GET("/domainlayer/repos", domainlayer.ReposIndex)

	r.GET("/plugininfo", plugininfo.Get)

	// mount all api resources for all plugins
	pluginsApiResources, err := services.GetPluginsApiResources()
	if err != nil {
		panic(err)
	}
	for pluginName, apiResources := range pluginsApiResources {
		for resourcePath, resourceHandlers := range apiResources {
			for method, h := range resourceHandlers {
				r.Handle(
					method,
					fmt.Sprintf("/plugins/%s/%s", pluginName, resourcePath),
					handlePluginCall(pluginName, h),
				)
			}
		}
	}
}

func handlePluginCall(pluginName string, handler core.ApiResourceHandler) func(c *gin.Context) {
	return func(c *gin.Context) {
		var err error
		input := &core.ApiResourceInput{}
		if len(c.Params) > 0 {
			input.Params = make(map[string]string)
			for _, param := range c.Params {
				input.Params[param.Key] = param.Value
			}
		}
		input.Query = c.Request.URL.Query()
		if c.Request.Body != nil {
			if strings.HasPrefix(c.Request.Header.Get("Content-Type"), "multipart/form-data;") {
				input.Request = c.Request
			} else {
				err = c.ShouldBindJSON(&input.Body)
				if err != nil && err.Error() != "EOF" {
					shared.ApiOutputError(c, err)
					return
				}
			}
		}
		output, err := handler(input)
		if err != nil {
			shared.ApiOutputError(c, err)
		} else if output != nil {
			status := output.Status
			if status < http.StatusContinue {
				status = http.StatusOK
			}
			if output.File != nil {
				c.Data(status, output.File.ContentType, output.File.Data)
				return
			}
			shared.ApiOutputSuccess(c, output.Body, status)
		} else {
			shared.ApiOutputSuccess(c, nil, http.StatusOK)
		}
	}
}
