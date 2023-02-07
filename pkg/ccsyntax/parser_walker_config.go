/*
Copyright 2023 Nokia.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ccsyntax

import (
	"fmt"

	ctrlcfgv1alpha1 "github.com/fnrunner/fnsyntax/apis/controllerconfig/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// cfgPreHookFn processes the for, own, watch generically
type cfgPreHookFn func(lcncCfg *ctrlcfgv1alpha1.ControllerConfig)
type cfgPostHookFn func(lcncCfg *ctrlcfgv1alpha1.ControllerConfig)

// gvkObjectFn processes the for, own, watch per item
type gvkObjectFn func(oc *OriginContext, v *ctrlcfgv1alpha1.GvkObject) *schema.GroupVersionKind
type emptyPipelineFn func(oc *OriginContext, v *ctrlcfgv1alpha1.GvkObject)

type pipelinePreHookFn func(oc *OriginContext, v *ctrlcfgv1alpha1.Pipeline)
type pipelinePostHookFn func(oc *OriginContext, v *ctrlcfgv1alpha1.Pipeline)

// functionFn processes the function in the functions section
type emptyFunctionElementFn func(oc *OriginContext)
type functionFn func(oc *OriginContext, v *ctrlcfgv1alpha1.Function)
type functionBlockFn func(oc *OriginContext, v *ctrlcfgv1alpha1.FunctionElement)

//type lcncServicesPreHookFn func(v []ctrlcfgv1.ControllerConfigFunctionsBlock)

//type lcncServicesPostHookFn func(v []LcncFunctionsBlock)

type serviceFn func(oc *OriginContext, v *ctrlcfgv1alpha1.Function)

// lcncServiceFn processes the service in the services section
//type lcncServiceFn func(o Origin, block bool, idx int, vertexName string, v ctrlcfgv1.ControllerConfigFunction)

type WalkConfig struct {
	cfgPreHookFn    cfgPreHookFn
	cfgPostHookFn   cfgPostHookFn
	gvkObjectFn     gvkObjectFn
	emptyPipelineFn emptyPipelineFn

	pipelinePreHookFn      pipelinePreHookFn
	emptyFunctionElementFn emptyFunctionElementFn
	functionBlockFn        functionBlockFn
	functionFn             functionFn
	pipelinePostHookFn     pipelinePostHookFn
	//lcncServicesPreHookFn   lcncServicesPreHookFn
	serviceFn serviceFn
	//lcncServicesPostHookFn  lcncServicesPreHookFn
}

func (r *parser) walkLcncConfig(fnc *WalkConfig) {
	// process config entry
	if fnc.cfgPreHookFn != nil {
		fnc.cfgPreHookFn(r.cCfg)
	}

	// process for, own, watch
	idx := 0
	for vertexName, v := range r.cCfg.GetFors() {
		// we run this once for apply and once for delete
		oc := &OriginContext{FOWS: FOWFor, RootVertexName: vertexName, Origin: OriginFow, VertexName: vertexName}
		r.processGvkObject(fnc, oc, v)
		idx++

	}
	idx = 0
	for vertexName, v := range r.cCfg.GetOwns() {
		// For Own the oepration is irrelevant
		oc := &OriginContext{FOWS: FOWOwn, RootVertexName: vertexName, Origin: OriginFow, VertexName: vertexName}
		r.processGvkObject(fnc, oc, v)
		idx++
	}
	idx = 0
	for vertexName, v := range r.cCfg.GetWatches() {
		// we run this only for operation apply, NOT for delete
		oc := &OriginContext{FOWS: FOWWatch, RootVertexName: vertexName, Origin: OriginFow, VertexName: vertexName}
		r.processGvkObject(fnc, oc, v)
	}

	if fnc.serviceFn != nil {
		fmt.Printf("services: %v\n", r.cCfg.GetServices())
		for vertexName, fn := range r.cCfg.GetServices() {
			oc := &OriginContext{FOWS: FOWService, RootVertexName: vertexName, Origin: OriginService, VertexName: vertexName}
			fnc.serviceFn(oc, fn)
		}
	}

	if fnc.cfgPostHookFn != nil {
		fnc.cfgPostHookFn(r.cCfg)
	}
}

func (r *parser) processGvkObject(fnc *WalkConfig, oc *OriginContext, v *ctrlcfgv1alpha1.GvkObject) {
	if fnc.gvkObjectFn != nil {
		gvk := fnc.gvkObjectFn(oc, v)
		oc.GVK = gvk
		oc.Operation = OperationApply
		applyPipeline := r.cCfg.GetPipeline(v.ApplyPipelineRef)
		if applyPipeline == nil {
			if fnc.emptyPipelineFn != nil {
				fnc.emptyPipelineFn(oc, v)
			}
		} else {
			fnc.walkPipeline(oc, applyPipeline)
		}

		oc.Operation = OperationDelete
		deletePipeline := r.cCfg.GetPipeline(v.DeletePipelineRef)
		if deletePipeline == nil {
			if fnc.emptyPipelineFn != nil {
				fnc.emptyPipelineFn(oc, v)
			}
		} else {
			fnc.walkPipeline(oc, deletePipeline)
		}
	}
}

func (fnc *WalkConfig) walkPipeline(oc *OriginContext, v *ctrlcfgv1alpha1.Pipeline) {
	pipelineName := v.Name
	if fnc.pipelinePreHookFn != nil {
		oc := &OriginContext{
			FOWS:           oc.FOWS,
			RootVertexName: oc.RootVertexName,
			Operation:      oc.Operation,
			GVK:            oc.GVK,
			Pipeline:       pipelineName,
			Origin:         oc.Origin,
			VertexName:     oc.VertexName,
		}
		fnc.pipelinePreHookFn(oc, v)
	}

	for vertexName, v := range v.Vars {
		oc := &OriginContext{
			FOWS:           oc.FOWS,
			RootVertexName: oc.RootVertexName,
			Operation:      oc.Operation,
			GVK:            oc.GVK,
			Pipeline:       pipelineName,
			Origin:         OriginVariable,
			VertexName:     vertexName,
			LocalVars:      v.Vars,
		}
		fnc.walkFunctionElement(oc, v)
	}

	for vertexName, v := range v.Tasks {
		oc := &OriginContext{
			FOWS:           oc.FOWS,
			RootVertexName: oc.RootVertexName,
			Operation:      oc.Operation,
			GVK:            oc.GVK,
			Pipeline:       pipelineName,
			Origin:         OriginFunction,
			VertexName:     vertexName,
			LocalVars:      v.Vars,
		}
		fnc.walkFunctionElement(oc, v)
	}

	if fnc.pipelinePostHookFn != nil {
		oc := &OriginContext{
			FOWS:           oc.FOWS,
			RootVertexName: oc.RootVertexName,
			Operation:      oc.Operation,
			GVK:            oc.GVK,
			Pipeline:       pipelineName,
			Origin:         oc.Origin,
			VertexName:     oc.VertexName,
		}
		fnc.pipelinePostHookFn(oc, v)
	}
}

func (fnc *WalkConfig) walkFunctionElement(oc *OriginContext, v *ctrlcfgv1alpha1.FunctionElement) {
	if v == nil {
		if fnc.emptyFunctionElementFn != nil {
			fnc.emptyFunctionElementFn(oc)
		}
		return
	}

	if v.Type == ctrlcfgv1alpha1.BlockType {
		if fnc.functionBlockFn != nil {
			// use to validate the function block
			fnc.functionBlockFn(oc, v)
		}
		// for a block function we allocate a new dag
		//if v.HasBlock() {
		//	oc.BlockDAG = dag.New()
		//}
		// the function in the function block is treated as a regular function
		if fnc.functionFn != nil {
			oc.Block = true
			fnc.functionFn(oc, &v.Function)
		}

		for vertexName, v := range v.FunctionBlock {
			oc := &OriginContext{
				FOWS:            oc.FOWS,
				RootVertexName:  oc.RootVertexName,
				Operation:       oc.Operation,
				GVK:             oc.GVK,
				Pipeline:        oc.Pipeline,
				Origin:          oc.Origin,
				Block:           true,
				BlockIndex:      oc.BlockIndex + 1,
				BlockVertexName: oc.VertexName,
				VertexName:      vertexName,
				LocalVars:       oc.LocalVars,
			}
			fnc.walkFunctionElement(oc, v)
		}
	} else {
		if fnc.functionFn != nil {
			fmt.Printf("oc function: %v\n", oc)
			fnc.functionFn(oc, &v.Function)
		}
	}
}
