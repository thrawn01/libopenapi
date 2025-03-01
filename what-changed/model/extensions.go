// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
    "github.com/pb33f/libopenapi/datamodel/low"
    "strings"
)

// ExtensionChanges represents any changes to custom extensions defined for an OpenAPI object.
type ExtensionChanges struct {
    *PropertyChanges
}

// GetAllChanges returns a slice of all changes made between Extension objects
func (e *ExtensionChanges) GetAllChanges() []*Change {
    return e.Changes
}

// TotalChanges returns the total number of object extensions that were made.
func (e *ExtensionChanges) TotalChanges() int {
    return e.PropertyChanges.TotalChanges()
}

// TotalBreakingChanges always returns 0 for Extension objects, they are non-binding.
func (e *ExtensionChanges) TotalBreakingChanges() int {
    return 0
}

// CompareExtensions will compare a left and right map of Tag/ValueReference models for any changes to
// anything. This function does not try and cast the value of an extension to perform checks, it
// will perform a basic value check.
//
// A current limitation relates to extensions being objects and a property of the object changes,
// there is currently no support for knowing anything changed - so it is ignored.
func CompareExtensions(l, r map[low.KeyReference[string]]low.ValueReference[any]) *ExtensionChanges {

    // look at the original and then look through the new.
    seenLeft := make(map[string]*low.ValueReference[any])
    seenRight := make(map[string]*low.ValueReference[any])
    for i := range l {
        h := l[i]
        seenLeft[strings.ToLower(i.Value)] = &h
    }
    for i := range r {
        h := r[i]
        seenRight[strings.ToLower(i.Value)] = &h
    }

    var changes []*Change
    for i := range seenLeft {

        CheckForObjectAdditionOrRemoval[any](seenLeft, seenRight, i, &changes, false, true)

        if seenRight[i] != nil {
            var props []*PropertyCheck

            props = append(props, &PropertyCheck{
                LeftNode:  seenLeft[i].ValueNode,
                RightNode: seenRight[i].ValueNode,
                Label:     i,
                Changes:   &changes,
                Breaking:  false,
                Original:  seenLeft[i].Value,
                New:       seenRight[i].Value,
            })

            // check properties
            CheckProperties(props)
        }
    }
    for i := range seenRight {
        if seenLeft[i] == nil {
            CheckForObjectAdditionOrRemoval[any](seenLeft, seenRight, i, &changes, false, true)
        }
    }
    ex := new(ExtensionChanges)
    ex.PropertyChanges = NewPropertyChanges(changes)
    if ex.TotalChanges() <= 0 {
        return nil
    }
    return ex
}

// CheckExtensions is a helper method to un-pack a left and right model that contains extensions. Once unpacked
// the extensions are compared and returns a pointer to ExtensionChanges. If nothing changed, nil is returned.
func CheckExtensions[T low.HasExtensions[T]](l, r T) *ExtensionChanges {
    var lExt, rExt map[low.KeyReference[string]]low.ValueReference[any]
    if len(l.GetExtensions()) > 0 {
        lExt = l.GetExtensions()
    }
    if len(r.GetExtensions()) > 0 {
        rExt = r.GetExtensions()
    }
    return CompareExtensions(lExt, rExt)
}
