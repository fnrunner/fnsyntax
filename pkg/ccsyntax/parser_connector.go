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
	"sync"

	"github.com/fnrunner/fnruntime/pkg/exec/rtdag"
	ctrlcfgv1alpha1 "github.com/fnrunner/fnsyntax/apis/controllerconfig/v1alpha1"
	"github.com/fnrunner/fnutils/pkg/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (r *parser) connect(ceCtx ConfigExecutionContext, gvar GlobalVariable) []Result {
	c := &connector{
		ceCtx:  ceCtx,
		gvar:   gvar,
		result: []Result{},
	}

	fnc := &WalkConfig{
		gvkObjectFn: c.connectGvk,
		functionFn:  c.connectFunction,
	}

	// walk the config resolve the verteces and create the outputmapping
	r.walkControllerConfig(fnc)
	// stop if errors were found
	return c.result
}

type connector struct {
	ceCtx  ConfigExecutionContext
	gvar   GlobalVariable
	mr     sync.RWMutex
	result []Result
}

func (r *connector) recordResult(result Result) {
	r.mr.Lock()
	defer r.mr.Unlock()
	r.result = append(r.result, result)
}

func (r *connector) connectGvk(oc *OriginContext, v *ctrlcfgv1alpha1.GvkObject) *schema.GroupVersionKind {
	gvk, err := meta.GetGVKFromRuntimeRawExtension(v.Resource)
	if err != nil {
		r.recordResult(Result{
			OriginContext: oc,
			Error:         err.Error(),
		})
	}
	return gvk
}

func (r *connector) connectFunction(oc *OriginContext, v *ctrlcfgv1alpha1.Function) {

	for localVarName, v := range v.Vars {
		oc.LocalVarName = localVarName
		r.connectRefs(oc, v)
	}

	if v.HasBlock() {
		r.connectBlock(oc, v.Block)
	}

	if v.Input != nil {
		if len(v.Input.Resource.Raw) != 0 {
			d := r.ceCtx.GetDAG(oc)
			// if the vertexName is within the block we need to connect to the root block vertex
			// otherwise we need to connect to the root Vertex
			if oc.BlockIndex > 0 {
				d.Connect(oc.BlockVertexName, oc.VertexName)
			} else {
				d.Connect(oc.RootVertexName, oc.VertexName)
			}
		}
		if v.Input.Key != "" {
			r.connectRefs(oc, v.Input.Key)
		}
		if v.Input.Value != "" {
			r.connectRefs(oc, v.Input.Value)
		}
		if v.Input.Expression != "" {
			r.connectRefs(oc, v.Input.Expression)
		}
		for _, v := range v.Input.GenericInput {
			r.connectRefs(oc, v)
		}
		if v.Input.Selector != nil {
			for k, v := range v.Input.Selector.MatchLabels {
				r.connectRefs(oc, k)
				r.connectRefs(oc, v)
			}
		}
	}

	// A block needs an explicit dependency to the root Block Vertex
	if oc.BlockIndex > 0 {
		r.connectVertex(oc, oc.BlockVertexName)
	}

	// A depndsOn needs an explicit dependency to the vertces they depend upon
	if len(v.DependsOn) != 0 {
		for _, vertexName := range v.DependsOn {
			r.connectVertex(oc, vertexName)
		}
	}
}

func (r *connector) connectBlock(oc *OriginContext, v ctrlcfgv1alpha1.Block) {
	if v.Range != nil {
		r.connectRefs(oc, v.Range.Value)
		// continue to resolve if this is a nested block
		if v.Range.Range != nil || v.Range.Condition != nil {
			r.connectBlock(oc, v.Range.Block)
		}
	}
	if v.Condition != nil {
		r.connectRefs(oc, v.Condition.Expression)
		// continue to resolve if this is a nested block
		if v.Condition.Range != nil || v.Condition.Condition != nil {
			r.connectBlock(oc, v.Condition.Block)
		}
	}
}

func (r *connector) connectRefs(oc *OriginContext, s string) {
	rfs := NewReferences()
	refs := rfs.GetReferences(s)

	for _, ref := range refs {
		// RangeRefKind do nothing
		// for regular values we resolve the variables
		// for variables that start with _ this is a special case and
		// should only be used within a jq construct
		if ref.Kind == RegularReferenceKind && ref.Value[0] != '_' {
			// get the vertexContext from the function
			//fmt.Printf("oc: %#v, ref: %#v, gvk: %s\n", oc, ref, oc.GVK.String())
			d := r.ceCtx.GetDAG(oc)
			if d == nil {
				fmt.Printf("empty dag\n")
			}
			//fmt.Printf("vertexName: %s\n", oc.VertexName)
			//d.PrintVertices()
			vc, ok := d.GetVertex(oc.VertexName).(*rtdag.VertexContext)
			if !ok {
				r.recordResult(Result{
					OriginContext: oc,
					Error:         fmt.Errorf("wrong type expect vertexContext: %#v", vc).Error(),
				})
			}
			//fmt.Printf("vc: %#v\n", vc)
			// lookup the localDAG first
			if oc.LocalVars != nil {
				//fmt.Printf("localVar: %#v\n", oc.LocalVars[ref.Value])
				if _, ok := oc.LocalVars[ref.Value]; ok {
					// add the local Variable to the reference list
					vc.AddReference(ref.Value)
					// if the lookup succeeds we are done
					continue
				}
			}
			// this is a global reference
			// the References are added to the runtime vertex context to ease processing
			vc.AddReference(ref.Value)
			varInfo := r.gvar.GetDAG(FOWEntry{FOW: oc.FOWS, RootVertexName: oc.RootVertexName}).GetVarInfo(ref.Value)
			if varInfo == nil {
				r.recordResult(Result{
					OriginContext: oc,
					Error:         fmt.Errorf("variable not found in gvar dag, varName: %s", ref.Value).Error(),
				})
				continue
			}

			switch {
			case varInfo.BlockIndex < oc.BlockIndex:
				//the reference points to the root DAG, so we need to wire it to the root of the block instead
				//fmt.Printf("connect: %s -> %s, oc: %#v, ref: %#v\n", oc.VertexName, varInfo.VertexName, oc, ref)
				d.Connect(oc.BlockVertexName, oc.VertexName)
				// connect the block vertex to the original vertex in the root DAG
				oc := oc.DeepCopy()
				oc.BlockIndex = 0
				oc.BlockVertexName = ""
				rootDAG := r.ceCtx.GetDAG(oc)
				rootDAG.Connect(varInfo.VertexName, oc.BlockVertexName)
			case varInfo.BlockIndex > oc.BlockIndex:
				// the reference points to an element in the block DAG, so we point to the root of the block
				d.Connect(varInfo.BlockVertexName, oc.VertexName)
			case varInfo.BlockIndex == oc.BlockIndex:
				d.Connect(varInfo.OutputVertex, oc.VertexName)
			}

		}
	}
}

func (r *connector) connectVertex(oc *OriginContext, vertexName string) {
	d := r.ceCtx.GetDAG(oc)
	d.Connect(vertexName, oc.VertexName)

}
