package resolver

import (
    "errors"
    "fmt"
    "io/ioutil"
    "net/url"
    "testing"

    "github.com/pb33f/libopenapi/index"
    "github.com/stretchr/testify/assert"
    "gopkg.in/yaml.v3"
)

func TestNewResolver(t *testing.T) {
    assert.Nil(t, NewResolver(nil))
}

func Benchmark_ResolveDocumentStripe(b *testing.B) {
    stripe, _ := ioutil.ReadFile("../test_specs/stripe.yaml")
    for n := 0; n < b.N; n++ {
        var rootNode yaml.Node
        yaml.Unmarshal(stripe, &rootNode)
        index := index.NewSpecIndex(&rootNode)
        resolver := NewResolver(index)
        resolver.Resolve()
    }
}

func TestResolver_ResolveComponents_CircularSpec(t *testing.T) {
    circular, _ := ioutil.ReadFile("../test_specs/circular-tests.yaml")
    var rootNode yaml.Node
    yaml.Unmarshal(circular, &rootNode)

    index := index.NewSpecIndex(&rootNode)

    resolver := NewResolver(index)
    assert.NotNil(t, resolver)

    circ := resolver.Resolve()
    assert.Len(t, circ, 3)

    _, err := yaml.Marshal(resolver.resolvedRoot)
    assert.NoError(t, err)
}

func TestResolver_CheckForCircularReferences(t *testing.T) {
    circular, _ := ioutil.ReadFile("../test_specs/circular-tests.yaml")
    var rootNode yaml.Node
    yaml.Unmarshal(circular, &rootNode)

    index := index.NewSpecIndex(&rootNode)

    resolver := NewResolver(index)
    assert.NotNil(t, resolver)

    circ := resolver.CheckForCircularReferences()
    assert.Len(t, circ, 3)
    assert.Len(t, resolver.GetResolvingErrors(), 3)
    assert.Len(t, resolver.GetCircularErrors(), 3)

    _, err := yaml.Marshal(resolver.resolvedRoot)
    assert.NoError(t, err)
}

func TestResolver_CheckForCircularReferences_DigitalOcean(t *testing.T) {
    circular, _ := ioutil.ReadFile("../test_specs/digitalocean.yaml")
    var rootNode yaml.Node
    yaml.Unmarshal(circular, &rootNode)

    baseURL, _ := url.Parse("https://raw.githubusercontent.com/digitalocean/openapi/main/specification")

    index := index.NewSpecIndexWithConfig(&rootNode, &index.SpecIndexConfig{
        AllowRemoteLookup: true,
        AllowFileLookup:   true,
        BaseURL:           baseURL,
    })

    resolver := NewResolver(index)
    assert.NotNil(t, resolver)

    circ := resolver.CheckForCircularReferences()
    assert.Len(t, circ, 0)
    assert.Len(t, resolver.GetResolvingErrors(), 0)
    assert.Len(t, resolver.GetCircularErrors(), 0)

    _, err := yaml.Marshal(resolver.resolvedRoot)
    assert.NoError(t, err)
}

func TestResolver_CircularReferencesRequiredValid(t *testing.T) {
    circular, _ := ioutil.ReadFile("../test_specs/swagger-valid-recursive-model.yaml")
    var rootNode yaml.Node
    yaml.Unmarshal(circular, &rootNode)

    index := index.NewSpecIndex(&rootNode)

    resolver := NewResolver(index)
    assert.NotNil(t, resolver)

    circ := resolver.CheckForCircularReferences()
    assert.Len(t, circ, 0)

    _, err := yaml.Marshal(resolver.resolvedRoot)
    assert.NoError(t, err)
}

func TestResolver_CircularReferencesRequiredInvalid(t *testing.T) {
    circular, _ := ioutil.ReadFile("../test_specs/swagger-invalid-recursive-model.yaml")
    var rootNode yaml.Node
    yaml.Unmarshal(circular, &rootNode)

    index := index.NewSpecIndex(&rootNode)

    resolver := NewResolver(index)
    assert.NotNil(t, resolver)

    circ := resolver.CheckForCircularReferences()
    assert.Len(t, circ, 2)

    _, err := yaml.Marshal(resolver.resolvedRoot)
    assert.NoError(t, err)
}

func TestResolver_DeepJourney(t *testing.T) {
    var journey []*index.Reference
    for f := 0; f < 200; f++ {
        journey = append(journey, nil)
    }
    index := index.NewSpecIndex(nil)
    resolver := NewResolver(index)
    assert.Nil(t, resolver.extractRelatives(nil, nil, journey, false))
}

func TestResolver_ResolveComponents_Stripe(t *testing.T) {
    stripe, _ := ioutil.ReadFile("../test_specs/stripe.yaml")
    var rootNode yaml.Node
    yaml.Unmarshal(stripe, &rootNode)

    index := index.NewSpecIndex(&rootNode)

    resolver := NewResolver(index)
    assert.NotNil(t, resolver)

    circ := resolver.Resolve()
    assert.Len(t, circ, 3)

    assert.Len(t, resolver.GetNonPolymorphicCircularErrors(), 3)
    assert.Len(t, resolver.GetPolymorphicCircularErrors(), 0)
}

func TestResolver_ResolveComponents_BurgerShop(t *testing.T) {
    mixedref, _ := ioutil.ReadFile("../test_specs/burgershop.openapi.yaml")
    var rootNode yaml.Node
    yaml.Unmarshal(mixedref, &rootNode)

    index := index.NewSpecIndex(&rootNode)

    resolver := NewResolver(index)
    assert.NotNil(t, resolver)

    circ := resolver.Resolve()
    assert.Len(t, circ, 0)
}

func TestResolver_ResolveComponents_PolyNonCircRef(t *testing.T) {
    yml := `paths:
  /hey:
    get:
      responses:
        "200":
          $ref: '#/components/schemas/crackers'
components:
  schemas:
    cheese:
      description: cheese
      anyOf:
        items:
          $ref: '#/components/schemas/crackers' 
    crackers:
      description: crackers
      allOf:
       - $ref: '#/components/schemas/tea'
    tea:
      description: tea`

    var rootNode yaml.Node
    yaml.Unmarshal([]byte(yml), &rootNode)

    index := index.NewSpecIndex(&rootNode)

    resolver := NewResolver(index)
    assert.NotNil(t, resolver)

    circ := resolver.CheckForCircularReferences()
    assert.Len(t, circ, 0)
}

func TestResolver_ResolveComponents_PolyCircRef(t *testing.T) {
    yml := `openapi: 3.1.0
components:
  schemas:
    cheese:
      description: cheese
      anyOf:
        - $ref: '#/components/schemas/crackers' 
    crackers:
      description: crackers
      anyOf:
       - $ref: '#/components/schemas/cheese'
    tea:
      description: tea`

    var rootNode yaml.Node
    yaml.Unmarshal([]byte(yml), &rootNode)

    index := index.NewSpecIndex(&rootNode)

    resolver := NewResolver(index)
    assert.NotNil(t, resolver)

    _ = resolver.CheckForCircularReferences()
    resolver.circularReferences[0].IsInfiniteLoop = true // override
    assert.Len(t, index.GetCircularReferences(), 1)
    assert.Len(t, resolver.GetPolymorphicCircularErrors(), 1)
    assert.Equal(t, 2, index.GetCircularReferences()[0].LoopIndex)

}

func TestResolver_ResolveComponents_Missing(t *testing.T) {
    yml := `paths:
  /hey:
    get:
      responses:
        "200":
          $ref: '#/components/schemas/crackers'
components:
  schemas:
    cheese:
      description: cheese
      properties:
        thang:
          $ref: '#/components/schemas/crackers' 
    crackers:
      description: crackers
      properties:
        butter:
          $ref: 'go home, I am drunk'`

    var rootNode yaml.Node
    yaml.Unmarshal([]byte(yml), &rootNode)

    index := index.NewSpecIndex(&rootNode)

    resolver := NewResolver(index)
    assert.NotNil(t, resolver)

    err := resolver.Resolve()
    assert.Len(t, err, 1)
    assert.Equal(t, "cannot resolve reference `go home, I am drunk`, it's missing: $go home, I am drunk [18:11]", err[0].Error())
}

func TestResolver_ResolveComponents_MixedRef(t *testing.T) {
    mixedref, _ := ioutil.ReadFile("../test_specs/mixedref-burgershop.openapi.yaml")
    var rootNode yaml.Node
    yaml.Unmarshal(mixedref, &rootNode)

    b := index.CreateOpenAPIIndexConfig()
    idx := index.NewSpecIndexWithConfig(&rootNode, b)

    resolver := NewResolver(idx)
    assert.NotNil(t, resolver)

    circ := resolver.Resolve()
    assert.Len(t, circ, 0)
    assert.Equal(t, 5, resolver.GetIndexesVisited())

    // in v0.8.2 a new check was added when indexing, to prevent re-indexing the same file multiple times.
    assert.Equal(t, 191, resolver.GetRelativesSeen())
    assert.Equal(t, 35, resolver.GetJourneysTaken())
    assert.Equal(t, 62, resolver.GetReferenceVisited())
}

func TestResolver_ResolveComponents_k8s(t *testing.T) {
    k8s, _ := ioutil.ReadFile("../test_specs/k8s.json")
    var rootNode yaml.Node
    yaml.Unmarshal(k8s, &rootNode)

    index := index.NewSpecIndex(&rootNode)

    resolver := NewResolver(index)
    assert.NotNil(t, resolver)

    circ := resolver.Resolve()
    assert.Len(t, circ, 0)
}

// Example of how to resolve the Stripe OpenAPI specification, and check for circular reference errors
func ExampleNewResolver() {
    // create a yaml.Node reference as a root node.
    var rootNode yaml.Node

    //  load in the Stripe OpenAPI spec (lots of polymorphic complexity in here)
    stripeBytes, _ := ioutil.ReadFile("../test_specs/stripe.yaml")

    // unmarshal bytes into our rootNode.
    _ = yaml.Unmarshal(stripeBytes, &rootNode)

    // create a new spec index (resolver depends on it)
    indexConfig := index.CreateClosedAPIIndexConfig()
    index := index.NewSpecIndexWithConfig(&rootNode, indexConfig)

    // create a new resolver using the index.
    resolver := NewResolver(index)

    // resolve the document, if there are circular reference errors, they are returned/
    // WARNING: this is a destructive action and the rootNode will be PERMANENTLY altered and cannot be unresolved
    circularErrors := resolver.Resolve()

    // The Stripe API has a bunch of circular reference problems, mainly from polymorphism.
    // So let's print them out.
    //
    fmt.Printf("There are %d circular reference errors, %d of them are polymorphic errors, %d are not",
        len(circularErrors), len(resolver.GetPolymorphicCircularErrors()), len(resolver.GetNonPolymorphicCircularErrors()))
    // Output: There are 3 circular reference errors, 0 of them are polymorphic errors, 3 are not
}

func ExampleResolvingError() {
    re := ResolvingError{
        ErrorRef: errors.New("Je suis une erreur"),
        Node: &yaml.Node{
            Line:   5,
            Column: 21,
        },
        Path:              "#/definitions/JeSuisUneErreur",
        CircularReference: &index.CircularReferenceResult{},
    }

    fmt.Printf("%s", re.Error())
    // Output: Je suis une erreur: #/definitions/JeSuisUneErreur [5:21]
}
