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

	ctrlcfgv1alpha1 "github.com/fnrunner/fnsyntax/apis/controllerconfig/v1alpha1"
)

func (r *parser) resolve(ceCtx ConfigExecutionContext, gvar GlobalVariable) []Result {
	rs := &resolver{
		ceCtx:  ceCtx,
		gvar:   gvar,
		result: []Result{},
	}

	fnc := &WalkConfig{
		//gvkObjectFn: rs.resolveGvk,
		functionFn: rs.resolveFunction,
	}

	// walk the config resolve the verteces and create the outputmapping
	r.walkControllerConfig(fnc)
	// stop if errors were found
	return rs.result
}

type resolver struct {
	ceCtx  ConfigExecutionContext
	gvar   GlobalVariable
	mr     sync.RWMutex
	result []Result
}

func (r *resolver) recordResult(result Result) {
	r.mr.Lock()
	defer r.mr.Unlock()
	r.result = append(r.result, result)
}

func (r *resolver) resolveFunction(oc *OriginContext, v *ctrlcfgv1alpha1.Function) {
	for localVarName, v := range v.Vars {
		oc.LocalVarName = localVarName
		r.resolveRefs(oc, v)
	}

	if v.HasBlock() {
		r.resolveBlock(oc, v.Block)
	}

	if v.Input.Selector != nil {
		for k, v := range v.Input.Selector.MatchLabels {
			r.resolveRefs(oc, k)
			r.resolveRefs(oc, v)
		}
	}

	if v.Input.Key != "" {
		r.resolveRefs(oc, v.Input.Key)
	}
	if v.Input.Value != "" {
		r.resolveRefs(oc, v.Input.Value)
	}
	if v.Input.Expression != "" {
		r.resolveRefs(oc, v.Input.Expression)
	}
	for _, v := range v.Input.GenericInput {
		r.resolveRefs(oc, v)
	}
	if len(v.DependsOn) > 0 {
		r.resolveDependsOn(oc, v.DependsOn)
	}
}

func (r *resolver) resolveBlock(oc *OriginContext, v ctrlcfgv1alpha1.Block) {
	if v.Range != nil {
		r.resolveRefs(oc, v.Range.Value)
		// continue to resolve if this is a nested block
		if v.Range.Range != nil || v.Range.Condition != nil {
			r.resolveBlock(oc, v.Range.Block)
		}

	}
	if v.Condition != nil {
		r.resolveRefs(oc, v.Condition.Expression)
		// continue to resolve if this is a nested block
		if v.Condition.Range != nil || v.Condition.Condition != nil {
			r.resolveBlock(oc, v.Condition.Block)
		}
	}

}

func (r *resolver) resolveRefs(oc *OriginContext, s string) {
	rfs := NewReferences()
	refs := rfs.GetReferences(s)

	for _, ref := range refs {
		// for regular values we resolve the variables
		// for variables that start with _ this is a special case and
		// should only be used within a jq construct
		if ref.Kind == RegularReferenceKind && ref.Value[0] != '_' {
			//d := r.ceCtx.GetDAG(oc)
			// get the vertexContext from the function
			//vc := d.GetVertex(oc.VertexName)
			// lookup the localDAG first
			if oc.LocalVars != nil {
				if _, ok := oc.LocalVars[ref.Value]; ok {
					// if the lookup succeeds we are done
					continue
				}
			}
			// we lookup in the outputDAG
			if !r.gvar.GetDAG(FOWEntry{FOW: oc.FOWS, RootVertexName: oc.RootVertexName}).VarExists(ref.Value) {
				r.recordResult(Result{
					OriginContext: oc,
					Error:         fmt.Errorf("cannot resolve %s", ref.Value).Error(),
				})
			}
		}
	}
}

func (r *resolver) resolveDependsOn(oc *OriginContext, vertexNames []string) {
	for _, vertexName := range vertexNames {
		if r.ceCtx.GetDAGCtx(oc.FOWS, oc.GVK, oc.Operation).DAG.GetVertex(vertexName) == nil {
			r.recordResult(Result{
				OriginContext: oc,
				Error:         fmt.Errorf("vertex in depndsOn does not exist %s", vertexName).Error(),
			})
		}
	}
}
