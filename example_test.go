package exps

import (
	"bytes"
	"fmt"
	"testing"
)

type RecordBenchmark struct {
	name string
	Nreq int `vals:"	30	00"`
	// Nkeys       int     `vals:"range(40000, 100000, 25000)"`
	Nsender     int     `vals:"1, 5"`
	GetRatio    float32 `vals:" 0, 4.3"`
	ReqType     string  `vals:" GET me,	 POST, PUT"`
	Record      bool
	result      interface{}
	privateData int
}

const expected = `name,Nreq,Nsender,GetRatio,ReqType,Record,result,privateData
,3000,1,0,GET me,true,123.12312312312312,0
,3000,1,0,GET me,false,<nil>,0
,3000,1,0,POST,true,<nil>,0
,3000,1,0,POST,false,<nil>,0
,3000,1,0,PUT,true,<nil>,0
,3000,1,0,PUT,false,<nil>,0
,3000,1,4.3,GET me,true,<nil>,0
,3000,1,4.3,GET me,false,<nil>,0
,3000,1,4.3,POST,true,<nil>,0
,3000,1,4.3,POST,false,<nil>,0
,3000,1,4.3,PUT,true,<nil>,0
,3000,1,4.3,PUT,false,<nil>,0
,3000,5,0,GET me,true,<nil>,0
,3000,5,0,GET me,false,<nil>,0
,3000,5,0,POST,true,<nil>,0
,3000,5,0,POST,false,<nil>,0
,3000,5,0,PUT,true,<nil>,0
,3000,5,0,PUT,false,<nil>,0
,3000,5,4.3,GET me,true,<nil>,0
,3000,5,4.3,GET me,false,<nil>,0
,3000,5,4.3,POST,true,<nil>,0
,3000,5,4.3,POST,false,<nil>,0
,3000,5,4.3,PUT,true,<nil>,0
,3000,5,4.3,PUT,false,<nil>,0
`

func MustEqual(expected string, got string) {
	if expected != got {
		panic(fmt.Sprintf("unmatched encodings expected:\n'%s' \ngot:\n'%s'", expected, got))
	}
}

func TestExperiment(t *testing.T) {
	// NUM_REQS := 3000
	buf := bytes.NewBuffer([]byte{})
	{
		expList := TemplateType[RecordBenchmark]()
		expList[0].result = 123.123123123123123
		ToCSVWriter(buf, expList)
		MustEqual(expected, buf.String())
	}

	buf.Reset()
	{
		expList := TemplateType[*RecordBenchmark]()
		expList[0].result = 123.123123123123123
		ToCSVWriter(buf, expList)
		MustEqual(expected, buf.String())
	}

	buf.Reset()
	{
		expList := Template(RecordBenchmark{})
		expList[0].result = 123.123123123123123
		ToCSVWriter(buf, expList)
		MustEqual(expected, buf.String())
	}

	buf.Reset()
	{
		expList := Template(&RecordBenchmark{})
		expList[0].result = 123.123123123123123
		ToCSVWriter(buf, expList)
		MustEqual(expected, buf.String())
	}

	buf.Reset()
	{
		expList := Template(&RecordBenchmark{})
		expList[0].result = 123.123123123123123
		ToCSVWriter(buf, expList)
		MustEqual(expected, buf.String())
	}
}

func TestNonDefaultPrivatePanic(t *testing.T) {
	defer func() { e := recover(); fmt.Println("expectedPanic:", e) }()
	bench := &RecordBenchmark{privateData: 42}
	Template(bench)
	t.Errorf("did not panic. should be illegal to pass private data to test generator")
}

type RecordArrayBenchmark struct {
	Nreq   int `vals:"1,2,3"`
	result []string
}

const expectedArr = `Nreq,result
1,
2,0
3,"0,1"
`

func TestArrayResultOutput(t *testing.T) {
	bench := &RecordArrayBenchmark{}
	exps := Template(bench)
	for i, exp := range exps {
		for j := 0; j < i; j++ {
			exp.result = append(exp.result, fmt.Sprint(j))
		}
	}
	buf := bytes.NewBuffer([]byte{})
	ToCSVWriter(buf, exps)
	MustEqual(expectedArr, buf.String())
}
