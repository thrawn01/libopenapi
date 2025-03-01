package low

import (
    "github.com/stretchr/testify/assert"
    "gopkg.in/yaml.v3"
    "sync"
    "testing"
)

type hotdog struct {
    Name            NodeReference[string]
    ValueName       ValueReference[string]
    Fat             NodeReference[int]
    Ketchup         NodeReference[float32]
    Mustard         NodeReference[float64]
    Grilled         NodeReference[bool]
    MaxTemp         NodeReference[int]
    MaxTempHigh     NodeReference[int64]
    MaxTempAlt      []NodeReference[int]
    Drinks          []NodeReference[string]
    Sides           []NodeReference[float32]
    BigSides        []NodeReference[float64]
    Temps           []NodeReference[int]
    HighTemps       []NodeReference[int64]
    Buns            []NodeReference[bool]
    UnknownElements NodeReference[any]
    LotsOfUnknowns  []NodeReference[any]
    Where           map[string]NodeReference[any]
    There           map[string]NodeReference[string]
    AllTheThings    NodeReference[map[KeyReference[string]]ValueReference[string]]
}

func TestBuildModel_Mismatch(t *testing.T) {

    yml := `crisps: are tasty`

    var rootNode yaml.Node
    mErr := yaml.Unmarshal([]byte(yml), &rootNode)
    assert.NoError(t, mErr)

    hd := hotdog{}
    cErr := BuildModel(&rootNode, &hd)
    assert.NoError(t, cErr)
    assert.Empty(t, hd.Name)

}

func TestBuildModel(t *testing.T) {

    yml := `name: yummy
valueName: yammy
beef: true
fat: 200
ketchup: 200.45
mustard: 324938249028.98234892374892374923874823974
grilled: true
maxTemp: 250
maxTempAlt: [1,2,3,4,5]
maxTempHigh: 7392837462032342
drinks:
  - nice
  - rice
  - spice
sides:
  - 0.23
  - 22.23
  - 99.45
  - 22311.2234
bigSides:
  - 98237498.9872349872349872349872347982734927342983479234234234234234234
  - 9827347234234.982374982734987234987
  - 234234234.234982374982347982374982374982347
  - 987234987234987234982734.987234987234987234987234987234987234987234982734982734982734987234987234987234987
temps: 
  - 1
  - 2
highTemps: 
  - 827349283744710
  - 11732849090192923
buns:
 - true
 - false
unknownElements:
  well:
    whoKnows: not me?
  doYou:
    love: beerToo?
lotsOfUnknowns:
  - wow:
      what: aTrip
  - amazing:
      french: fries
  - amazing:
      french: fries
where:
  things:
    are:
      wild: out here
  howMany:
    bears: 200
there:
  oh: yeah
  care: bear
allTheThings:
  beer: isGood
  cake: isNice`

    var rootNode yaml.Node
    mErr := yaml.Unmarshal([]byte(yml), &rootNode)
    assert.NoError(t, mErr)

    hd := hotdog{}
    cErr := BuildModel(rootNode.Content[0], &hd)
    assert.Equal(t, 200, hd.Fat.Value)
    assert.Equal(t, 4, hd.Fat.ValueNode.Line)
    assert.Equal(t, true, hd.Grilled.Value)
    assert.Equal(t, "yummy", hd.Name.Value)
    assert.Equal(t, "yammy", hd.ValueName.Value)
    assert.Equal(t, float32(200.45), hd.Ketchup.Value)
    assert.Len(t, hd.Drinks, 3)
    assert.Len(t, hd.Sides, 4)
    assert.Len(t, hd.BigSides, 4)
    assert.Len(t, hd.Temps, 2)
    assert.Len(t, hd.HighTemps, 2)
    assert.Equal(t, int64(11732849090192923), hd.HighTemps[1].Value)
    assert.Len(t, hd.MaxTempAlt, 5)
    assert.Equal(t, int64(7392837462032342), hd.MaxTempHigh.Value)
    assert.Equal(t, 2, hd.Temps[1].Value)
    assert.Equal(t, 27, hd.Temps[1].ValueNode.Line)
    assert.Len(t, hd.UnknownElements.Value, 2)
    assert.Len(t, hd.LotsOfUnknowns, 3)
    assert.Len(t, hd.Where, 2)
    assert.Len(t, hd.There, 2)
    assert.Equal(t, "bear", hd.There["care"].Value)
    assert.Equal(t, 324938249028.98234892374892374923874823974, hd.Mustard.Value)

    allTheThings := hd.AllTheThings.Value
    for i := range allTheThings {
        if i.Value == "beer" {
            assert.Equal(t, "isGood", allTheThings[i].Value)
        }
        if i.Value == "cake" {
            assert.Equal(t, "isNice", allTheThings[i].Value)
        }
    }
    assert.NoError(t, cErr)
}

func TestBuildModel_UseCopyNotRef(t *testing.T) {

    yml := `cake: -99999`

    var rootNode yaml.Node
    mErr := yaml.Unmarshal([]byte(yml), &rootNode)
    assert.NoError(t, mErr)

    hd := hotdog{}
    cErr := BuildModel(&rootNode, hd)
    assert.Error(t, cErr)
    assert.Empty(t, hd.Name)

}

func TestBuildModel_UseUnsupportedPrimitive(t *testing.T) {

    type notSupported struct {
        cake string
    }
    ns := notSupported{}
    yml := `cake: party`

    var rootNode yaml.Node
    mErr := yaml.Unmarshal([]byte(yml), &rootNode)
    assert.NoError(t, mErr)

    cErr := BuildModel(rootNode.Content[0], &ns)
    assert.Error(t, cErr)
    assert.Empty(t, ns.cake)

}

func TestBuildModel_UsingInternalConstructs(t *testing.T) {

    type internal struct {
        Extensions NodeReference[string]
        PathItems  NodeReference[string]
        Thing      NodeReference[string]
    }

    yml := `extensions: one
pathItems: two
thing: yeah`

    ins := new(internal)
    var rootNode yaml.Node
    mErr := yaml.Unmarshal([]byte(yml), &rootNode)
    assert.NoError(t, mErr)

    // try a null build
    try := BuildModel(nil, ins)
    assert.NoError(t, try)

    cErr := BuildModel(rootNode.Content[0], ins)
    assert.NoError(t, cErr)
    assert.Empty(t, ins.PathItems.Value)
    assert.Empty(t, ins.Extensions.Value)
    assert.Equal(t, "yeah", ins.Thing.Value)
}

func TestSetField_NodeRefAny_Error(t *testing.T) {

    type internal struct {
        Thing []NodeReference[any]
    }

    yml := `thing:
  - 999
  - false`

    ins := new(internal)
    var rootNode yaml.Node
    mErr := yaml.Unmarshal([]byte(yml), &rootNode)
    assert.NoError(t, mErr)

    try := BuildModel(rootNode.Content[0], ins)
    assert.Error(t, try)

}

func TestSetField_MapHelperWrapped(t *testing.T) {

    type internal struct {
        Thing KeyReference[map[KeyReference[string]]ValueReference[string]]
    }

    yml := `thing: 
  what: not
  chip: chop
  lip: lop`

    ins := new(internal)
    var rootNode yaml.Node
    mErr := yaml.Unmarshal([]byte(yml), &rootNode)
    assert.NoError(t, mErr)

    try := BuildModel(rootNode.Content[0], ins)
    assert.NoError(t, try)
    assert.Len(t, ins.Thing.Value, 3)
}

func TestSetField_MapHelper(t *testing.T) {

    type internal struct {
        Thing map[KeyReference[string]]ValueReference[string]
    }

    yml := `thing: 
  what: not
  chip: chop
  lip: lop`

    ins := new(internal)
    var rootNode yaml.Node
    mErr := yaml.Unmarshal([]byte(yml), &rootNode)
    assert.NoError(t, mErr)

    try := BuildModel(rootNode.Content[0], ins)
    assert.NoError(t, try)
    assert.Len(t, ins.Thing, 3)
}

func TestSetField_ArrayHelper(t *testing.T) {

    type internal struct {
        Thing NodeReference[[]ValueReference[string]]
    }

    yml := `thing: 
  - nice
  - rice
  - slice`

    ins := new(internal)
    var rootNode yaml.Node
    mErr := yaml.Unmarshal([]byte(yml), &rootNode)
    assert.NoError(t, mErr)

    try := BuildModel(rootNode.Content[0], ins)
    assert.NoError(t, try)
    assert.Len(t, ins.Thing.Value, 3)
}

func TestSetField_Enum_Helper(t *testing.T) {

    type internal struct {
        Thing NodeReference[[]ValueReference[any]]
    }

    yml := `thing: 
  - nice
  - rice
  - slice`

    ins := new(internal)
    var rootNode yaml.Node
    mErr := yaml.Unmarshal([]byte(yml), &rootNode)
    assert.NoError(t, mErr)

    try := BuildModel(rootNode.Content[0], ins)
    assert.NoError(t, try)
    assert.Len(t, ins.Thing.Value, 3)
}

func TestSetField_Default_Helper(t *testing.T) {

    type cake struct {
        thing int
    }

    // this should be ignored, no custom objects in here my friend.
    type internal struct {
        Thing cake
    }

    yml := `thing: 
  type: cake`

    ins := new(internal)
    var rootNode yaml.Node
    mErr := yaml.Unmarshal([]byte(yml), &rootNode)
    assert.NoError(t, mErr)

    try := BuildModel(rootNode.Content[0], ins)
    assert.NoError(t, try)
    assert.Equal(t, 0, ins.Thing.thing)
}

func TestHandleSlicesOfInts(t *testing.T) {

    type internal struct {
        Thing NodeReference[[]ValueReference[any]]
    }

    yml := `thing:
  - 5
  - 1.234`

    ins := new(internal)
    var rootNode yaml.Node
    mErr := yaml.Unmarshal([]byte(yml), &rootNode)
    assert.NoError(t, mErr)

    try := BuildModel(rootNode.Content[0], ins)
    assert.NoError(t, try)
    assert.Equal(t, int64(5), ins.Thing.Value[0].Value)
    assert.Equal(t, 1.234, ins.Thing.Value[1].Value)
}

func TestHandleSlicesOfBools(t *testing.T) {
    type internal struct {
        Thing NodeReference[[]ValueReference[any]]
    }

    yml := `thing:
  - true
  - false`

    ins := new(internal)
    var rootNode yaml.Node
    mErr := yaml.Unmarshal([]byte(yml), &rootNode)
    assert.NoError(t, mErr)

    try := BuildModel(rootNode.Content[0], ins)
    assert.NoError(t, try)
    assert.Equal(t, true, ins.Thing.Value[0].Value)
    assert.Equal(t, false, ins.Thing.Value[1].Value)
}

func TestSetField_Ignore(t *testing.T) {

    type Complex struct {
        name string
    }
    type internal struct {
        Thing *Complex
    }

    yml := `thing: 
  - nice
  - rice
  - slice`

    ins := new(internal)
    var rootNode yaml.Node
    mErr := yaml.Unmarshal([]byte(yml), &rootNode)
    assert.NoError(t, mErr)

    try := BuildModel(&rootNode, ins)
    assert.NoError(t, try)
    assert.Nil(t, ins.Thing)
}

func TestBuildModelAsync(t *testing.T) {

    type internal struct {
        Thing KeyReference[map[KeyReference[string]]ValueReference[string]]
    }

    yml := `thing: 
  what: not
  chip: chop
  lip: lop`

    ins := new(internal)
    var rootNode yaml.Node
    mErr := yaml.Unmarshal([]byte(yml), &rootNode)
    assert.NoError(t, mErr)

    var wg sync.WaitGroup
    var errors []error
    wg.Add(1)
    BuildModelAsync(rootNode.Content[0], ins, &wg, &errors)
    wg.Wait()
    assert.Len(t, ins.Thing.Value, 3)

}

func TestBuildModelAsync_Error(t *testing.T) {

    type internal struct {
        Thing []NodeReference[any]
    }

    yml := `thing:
  - 999
  - false`

    ins := new(internal)
    var rootNode yaml.Node
    mErr := yaml.Unmarshal([]byte(yml), &rootNode)
    assert.NoError(t, mErr)

    var wg sync.WaitGroup
    var errors []error
    wg.Add(1)
    BuildModelAsync(rootNode.Content[0], ins, &wg, &errors)
    wg.Wait()
    assert.Len(t, errors, 1)
    assert.Len(t, ins.Thing, 0)

}
