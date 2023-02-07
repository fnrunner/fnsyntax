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

	"github.com/fnrunner/fnsyntax/pkg/ccsyntax/vardag"
)

// GlobalVariable stores the variable context in a global DAG for validating
// that the variables are globally unique. This is only used by the parser
// for resolving and connecting the runtime graph
type GlobalVariable interface {
	GetName() string
	Add(fe FOWEntry)
	GetDAG(fe FOWEntry) vardag.VarDAG
	Print()
}

type VariableContext struct {
	name string
	m    sync.RWMutex
	o    map[FOWEntry]vardag.VarDAG
}

type FOWEntry struct {
	FOW            FOWS
	RootVertexName string
}

func NewGlobalVariable(n string) GlobalVariable {
	return &VariableContext{
		name: n,
		o:    make(map[FOWEntry]vardag.VarDAG),
	}
}

func (r *VariableContext) GetName() string {
	return r.name
}

func (r *VariableContext) Add(fe FOWEntry) {
	r.m.Lock()
	defer r.m.Unlock()
	if _, ok := r.o[fe]; !ok {
		r.o[fe] = vardag.New()
	}
}

func (r *VariableContext) GetDAG(fe FOWEntry) vardag.VarDAG {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.o[fe]
}

func (r *VariableContext) Print() {
	fmt.Printf("Name: %s\n", r.name)
	for fe, d := range r.o {
		fmt.Printf("FOW: %s, RootVertexname: %s\n", fe.FOW, fe.RootVertexName)
		d.Print()
	}
}
