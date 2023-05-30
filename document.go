// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

// Package libopenapi is a library containing tools for reading and in and manipulating Swagger (OpenAPI 2) and OpenAPI 3+
// specifications into strongly typed documents. These documents have two APIs, a high level (porcelain) and a
// low level (plumbing).
//
// Every single type has a 'GoLow()' method that drops down from the high API to the low API. Once in the low API,
// the entire original document data is available, including all comments, line and column numbers for keys and values.
//
// There are two steps to creating a using Document. First, create a new Document using the NewDocument() method
// and pass in a specification []byte array that contains the OpenAPI Specification. It doesn't matter if YAML or JSON
// are used.
package libopenapi

import (
	"fmt"

	"github.com/pb33f/libopenapi/index"

	"github.com/pb33f/libopenapi/datamodel"
	v2high "github.com/pb33f/libopenapi/datamodel/high/v2"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
	v2low "github.com/pb33f/libopenapi/datamodel/low/v2"
	v3low "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/resolver"
	"github.com/pb33f/libopenapi/utils"
	what_changed "github.com/pb33f/libopenapi/what-changed"
	"github.com/pb33f/libopenapi/what-changed/model"
	"gopkg.in/yaml.v3"
)

// Document Represents an OpenAPI specification that can then be rendered into a model or serialized back into
// a string document after being manipulated.
type Document interface {
	// GetVersion will return the exact version of the OpenAPI specification set for the document.
	GetVersion() string

	// GetSpecInfo will return the *datamodel.SpecInfo instance that contains all specification information.
	GetSpecInfo() *datamodel.SpecInfo

	// SetConfiguration will set the configuration for the document. This allows for finer grained control over
	// allowing remote or local references, as well as a BaseURL to allow for relative file references.
	SetConfiguration(configuration *datamodel.DocumentConfiguration)

	// BuildV2Model will build out a Swagger (version 2) model from the specification used to create the document
	// If there are any issues, then no model will be returned, instead a slice of errors will explain all the
	// problems that occurred. This method will only support version 2 specifications and will throw an error for
	// any other types.
	BuildV2Model() (*DocumentModel[v2high.Swagger], error)

	// BuildV3Model will build out an OpenAPI (version 3+) model from the specification used to create the document
	// If there are any issues, then no model will be returned, instead a slice of errors will explain all the
	// problems that occurred. This method will only support version 3 specifications and will throw an error for
	// any other types.
	BuildV3Model() (*DocumentModel[v3high.Document], error)

	// RenderAndReload will render the high level model as it currently exists (including any mutations, additions
	// and removals to and from any object in the tree). It will then reload the low level model with the new bytes
	// extracted from the model that was re-rendered. This is useful if you want to make changes to the high level model
	// and then 'reload' the model into memory, so that line numbers and column numbers are correct and all update
	// according to the changes made.
	//
	// The method returns the raw YAML bytes that were rendered, and any errors that occurred during rebuilding of the model.
	// This is a destructive operation, and will re-build the entire model from scratch using the new bytes, so any
	// references to the old model will be lost. The second return is the new Document that was created, and the third
	// return is any errors hit trying to re-render.
	//
	// **IMPORTANT** This method only supports OpenAPI Documents. The Swagger model will not support mutations correctly
	// and will not update when called. This choice has been made because we don't want to continue supporting Swagger,
	// it's too old, so it should be motivation to upgrade to OpenAPI 3.
	RenderAndReload() ([]byte, Document, *DocumentModel[v3high.Document], error)

	// Serialize will re-render a Document back into a []byte slice. If any modifications have been made to the
	// underlying data model using low level APIs, then those changes will be reflected in the serialized output.
	//
	// It's important to know that this should not be used if the resolver has been used on a specification to
	// for anything other than checking for circular references. If the resolver is used to resolve the spec, then this
	// method may spin out forever if the specification backing the model has circular references.
	// Deprecated: This method is deprecated and will be removed in a future release. Use RenderAndReload() instead.
	// This method does not support mutations correctly.
	Serialize() ([]byte, error)
}

type document struct {
	version           string
	info              *datamodel.SpecInfo
	config            *datamodel.DocumentConfiguration
	highOpenAPI3Model *DocumentModel[v3high.Document]
	highSwaggerModel  *DocumentModel[v2high.Swagger]
}

// DocumentModel represents either a Swagger document (version 2) or an OpenAPI document (version 3) that is
// built from a parent Document.
type DocumentModel[T v2high.Swagger | v3high.Document] struct {
	Model T
	Index *index.SpecIndex // index created from the document.
}

// NewDocument will create a new OpenAPI instance from an OpenAPI specification []byte array. If anything goes
// wrong when parsing, reading or processing the OpenAPI specification, there will be no document returned, instead
// a slice of errors will be returned that explain everything that failed.
//
// After creating a Document, the option to build a model becomes available, in either V2 or V3 flavors. The models
// are about 70% different between Swagger and OpenAPI 3, which is why two different models are available.
//
// This function will automatically follow (meaning load) any file or remote references that are found anywhere.
// Which means recursively also, like a spider, it will follow every reference found, local or remote.
//
// If this isn't the behavior you want, or that you feel this is a potential security risk,
// then you can use the NewDocumentWithConfiguration() function instead, which allows you to set a configuration that
// will allow you to control if file or remote references are allowed.
func NewDocument(specByteArray []byte) (Document, error) {
	info, err := datamodel.ExtractSpecInfo(specByteArray)
	if err != nil {
		return nil, err
	}
	d := new(document)
	d.version = info.Version
	d.info = info
	return d, nil
}

// NewDocumentWithConfiguration is the same as NewDocument, except it's a convenience function that calls NewDocument
// under the hood and then calls SetConfiguration() on the returned Document.
func NewDocumentWithConfiguration(specByteArray []byte, configuration *datamodel.DocumentConfiguration) (Document, error) {
	d, err := NewDocument(specByteArray)
	if d != nil {
		d.SetConfiguration(configuration)
	}
	return d, err
}

func (d *document) GetVersion() string {
	return d.version
}

func (d *document) GetSpecInfo() *datamodel.SpecInfo {
	return d.info
}

func (d *document) SetConfiguration(configuration *datamodel.DocumentConfiguration) {
	d.config = configuration
}

func (d *document) Serialize() ([]byte, error) {
	if d.info == nil {
		return nil, fmt.Errorf("unable to serialize, document has not yet been initialized")
	}
	if d.info.SpecFileType == datamodel.YAMLFileType {
		return yaml.Marshal(d.info.RootNode)
	} else {
		yamlData, _ := yaml.Marshal(d.info.RootNode)
		return utils.ConvertYAMLtoJSON(yamlData)
	}
}

func (d *document) RenderAndReload() ([]byte, Document, *DocumentModel[v3high.Document], error) {
	if d.highSwaggerModel != nil && d.highOpenAPI3Model == nil {
		return nil, nil, nil, errorMsg("this method only supports OpenAPI 3 documents, not Swagger")
	}

	var newBytes []byte

	// render the model as the correct type based on the source.
	// https://github.com/pb33f/libopenapi/issues/105
	if d.info.SpecFileType == datamodel.JSONFileType {
		jsonIndent := "  "
		i := d.info.OriginalIndentation
		if i > 2 {
			for l := 0; l < i-2; l++ {
				jsonIndent += " "
			}
		}
		newBytes = d.highOpenAPI3Model.Model.RenderJSON(jsonIndent)
	}
	if d.info.SpecFileType == datamodel.YAMLFileType {
		newBytes = d.highOpenAPI3Model.Model.RenderWithIndention(d.info.OriginalIndentation)
	}

	newDoc, err := NewDocumentWithConfiguration(newBytes, d.config)
	if err != nil {
		return newBytes, newDoc, nil, wrapErr(err)
	}
	// build the model.
	model, err := newDoc.BuildV3Model()
	if err != nil {
		return newBytes, newDoc, model, wrapErr(err)
	}
	// this document is now dead, long live the new document!
	return newBytes, newDoc, model, nil
}

func (d *document) BuildV2Model() (*DocumentModel[v2high.Swagger], error) {
	if d.highSwaggerModel != nil {
		return d.highSwaggerModel, nil
	}
	if d.info == nil {
		return nil, errorMsg("unable to build swagger document, no specification has been loaded")
	}
	if d.info.SpecFormat != datamodel.OAS2 {
		return nil, errorMsgf("unable to build swagger document, "+
			"supplied spec is a different version (%v). Try 'BuildV3Model()'", d.info.SpecFormat)
	}

	var lowDoc *v2low.Swagger
	if d.config == nil {
		d.config = &datamodel.DocumentConfiguration{
			AllowFileReferences:   false,
			AllowRemoteReferences: false,
		}
	}

	lowDoc, errs := v2low.CreateDocumentFromConfig(d.info, d.config)
	// Do not short-circuit on circular reference errors, so the client
	// has the option of ignoring them.
	for _, err := range errs {
		if refErr, ok := err.(*resolver.ResolvingError); ok {
			if refErr.CircularReference == nil {
				return nil, wrapErrs(errs)
			}
		} else {
			return nil, wrapErrs(errs)
		}
	}
	highDoc := v2high.NewSwaggerDocument(lowDoc)
	d.highSwaggerModel = &DocumentModel[v2high.Swagger]{
		Model: *highDoc,
		Index: lowDoc.Index,
	}
	return d.highSwaggerModel, wrapErrs(errs)
}

func (d *document) BuildV3Model() (*DocumentModel[v3high.Document], error) {
	if d.highOpenAPI3Model != nil {
		return d.highOpenAPI3Model, nil
	}
	if d.info == nil {
		return nil, errorMsg("unable to build document, no specification has been loaded")
	}
	if d.info.SpecFormat != datamodel.OAS3 {
		return nil, errorMsgf("unable to build openapi document, "+
			"supplied spec is a different version (%v). Try 'BuildV2Model()'", d.info.SpecFormat)
	}

	var lowDoc *v3low.Document
	if d.config == nil {
		d.config = &datamodel.DocumentConfiguration{
			AllowFileReferences:   false,
			AllowRemoteReferences: false,
		}
	}

	lowDoc, errs := v3low.CreateDocumentFromConfig(d.info, d.config)
	// Do not short-circuit on circular reference errors, so the client
	// has the option of ignoring them.
	for _, err := range errs {
		if refErr, ok := err.(*resolver.ResolvingError); ok {
			if refErr.CircularReference == nil {
				return nil, wrapErrs(errs)
			}
		} else {
			return nil, wrapErrs(errs)
		}
	}
	highDoc := v3high.NewDocument(lowDoc)
	d.highOpenAPI3Model = &DocumentModel[v3high.Document]{
		Model: *highDoc,
		Index: lowDoc.Index,
	}
	return d.highOpenAPI3Model, wrapErrs(errs)
}

// CompareDocuments will accept a left and right Document implementing struct, build a model for the correct
// version and then compare model documents for changes.
//
// If there are any errors when building the models, those errors are returned with a nil pointer for the
// model.DocumentChanges. If there are any changes found however between either Document, then a pointer to
// model.DocumentChanges is returned containing every single change, broken down, model by model.
func CompareDocuments(original, updated Document) (*model.DocumentChanges, error) {
	errs := &MultiError{}
	if original.GetSpecInfo().SpecType == utils.OpenApi3 && updated.GetSpecInfo().SpecType == utils.OpenApi3 {
		v3ModelLeft, err := original.BuildV3Model()
		if err != nil {
			errs.Append(err)
		}
		v3ModelRight, err := updated.BuildV3Model()
		if err != nil {
			errs.Append(err)
		}
		if v3ModelLeft != nil && v3ModelRight != nil {
			return what_changed.CompareOpenAPIDocuments(v3ModelLeft.Model.GoLow(), v3ModelRight.Model.GoLow()),
				errs.OrNil()
		} else {
			return nil, errs.OrNil()
		}
	}
	if original.GetSpecInfo().SpecType == utils.OpenApi2 && updated.GetSpecInfo().SpecType == utils.OpenApi2 {
		errs := &MultiError{}
		v2ModelLeft, err := original.BuildV2Model()
		if err != nil {
			errs.Append(err)
		}
		v2ModelRight, err := updated.BuildV2Model()
		if err != nil {
			errs.Append(err)
		}
		if v2ModelLeft != nil && v2ModelRight != nil {
			return what_changed.CompareSwaggerDocuments(v2ModelLeft.Model.GoLow(), v2ModelRight.Model.GoLow()),
				errs.OrNil()
		} else {
			return nil, errs.OrNil()
		}
	}
	return nil, errorMsg("unable to compare documents, one or both documents are not of the same version")
}
