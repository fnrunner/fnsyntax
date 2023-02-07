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
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Result struct {
	OriginContext *OriginContext `json:"inline" yaml:"inline"`
	Error         string         `json:"error,omitempty" yaml:"error,omitempty"`
}

type recordResultFn func(Result)

type OriginContext struct {
	//Index      int
	FOWS            FOWS                     `json:"fow,omitempty" yaml:"fow,omitempty"`
	RootVertexName  string                   `json:"rootVertexName,omitempty" yaml:"rootVertexName,omitempty"`
	GVK             *schema.GroupVersionKind `json:"gvk,omitempty" yaml:"gvk,omitempty"`
	Operation       Operation                `json:"operation,omitempty" yaml:"operation,omitempty"`
	Pipeline        string                   `json:"pipeline,omitempty" yaml:"pipeline,omitempty"`
	Origin          Origin                   `json:"origin,omitempty" yaml:"origin,omitempty"`
	Block           bool                     `json:"block,omitempty" yaml:"block,omitempty"`
	BlockIndex      int                      `json:"blockIdx,omitempty" yaml:"blockIdx,omitempty"`
	BlockVertexName string                   `json:"blockVertexName,omitempty" yaml:"blockVertexName,omitempty"`
	VertexName      string                   `json:"vertexname,omitempty" yaml:"vertexname,omitempty"`
	LocalVarName    string                   `json:"localvarName,omitempty" yaml:"localvarName,omitempty"`
	LocalVars       map[string]string        `json:"localVars,omitempty" yaml:"localvarName,omitempty"`
}

func (in *OriginContext) DeepCopy() *OriginContext {
	if in == nil {
		return nil
	}
	out := new(OriginContext)
	in.DeepCopyInto(out)
	return out
}

func (in *OriginContext) DeepCopyInto(out *OriginContext) {
	*out = *in
}

type FOWS string

const (
	FOWFor     FOWS = "for"
	FOWOwn     FOWS = "own"
	FOWWatch   FOWS = "watch"
	FOWService FOWS = "service"
)

type Operation string

const (
	OperationApply  Operation = "apply"
	OperationDelete Operation = "delete"
	OperationNone   Operation = "none"
)

type Origin string

const (
	OriginInvalid  Origin = "invalid"
	OriginFow      Origin = "fow"
	OriginService  Origin = "service"
	OriginVariable Origin = "vars"
	OriginFunction Origin = "function"
)
