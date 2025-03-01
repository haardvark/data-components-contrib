package csv

import (
	"bytes"
	"io"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/bradleyjkemp/cupaloy"
	"github.com/spiceai/data-components-contrib/dataconnectors/file"
	"github.com/spiceai/spiceai/pkg/observations"
	"github.com/stretchr/testify/assert"
)

var snapshotter = cupaloy.New(cupaloy.SnapshotSubdirectory("../../test/assets/snapshots/dataprocessors/csv"))

func TestCsv(t *testing.T) {
	epoch := time.Unix(1605312000, 0)
	period := 7 * 24 * time.Hour
	interval := time.Hour

	var wg sync.WaitGroup

	localFileConnector := file.NewFileConnector()

	var localData []byte
	err := localFileConnector.Read(func(data []byte, metadata map[string]string) ([]byte, error) {
		localData = data
		wg.Done()
		return nil, nil
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	wg.Add(1)

	err = localFileConnector.Init(epoch, period, interval, map[string]string{
		"path":  "../../test/assets/data/csv/COINBASE_BTCUSD, 30.csv",
		"watch": "false",
	})
	if err != nil {
		t.Fatal(err.Error())
	}

	localDataTags, err := os.ReadFile("../../test/assets/data/csv/local_tag_data.csv")
	if err != nil {
		t.Fatal(err.Error())
	}

	globalDataTags, err := os.ReadFile("../../test/assets/data/csv/global_tag_data.csv")
	if err != nil {
		t.Fatal(err.Error())
	}

	globalFileConnector := file.NewFileConnector()

	var globalData []byte
	err = globalFileConnector.Read(func(data []byte, metadata map[string]string) ([]byte, error) {
		globalData = data
		wg.Done()
		return nil, nil
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	wg.Add(1)

	err = globalFileConnector.Init(epoch, period, interval, map[string]string{
		"path":  "../../test/assets/data/csv/trader_input.csv",
		"watch": "false",
	})
	if err != nil {
		t.Fatal(err.Error())
	}

	wg.Wait()

	t.Run("Init()", testInitFunc())
	t.Run("GetObservations()", testGetObservationsFunc(localData))
	t.Run("GetObservations() custom time format", testGetObservationsCustomTimeFunc())
	t.Run("GetObservations() with tags", testGetObservationsFunc(localDataTags))
	t.Run("GetObservations() called twice", testGetObservationsTwiceFunc(localData))
	t.Run("GetObservations() updated with same data", testGetObservationsSameDataFunc(localData))
	t.Run("GetState()", testGetStateFunc(globalData))
	t.Run("GetState() with tags", testGetStateTagsFunc(globalDataTags))
	t.Run("GetState() called twice", testGetStateTwiceFunc(globalData))
	t.Run("getColumnMappings()", testgetColumnMappingsFunc())
}

func BenchmarkGetObservations(b *testing.B) {
	epoch := time.Unix(1605312000, 0)
	period := 7 * 24 * time.Hour
	interval := time.Hour

	localFileConnector := file.NewFileConnector()

	err := localFileConnector.Read(func(data []byte, metadata map[string]string) ([]byte, error) {
		return nil, nil
	})
	if err != nil {
		b.Fatal(err.Error())
	}

	err = localFileConnector.Init(epoch, period, interval, map[string]string{
		"path":  "../../test/assets/data/csv/COINBASE_BTCUSD, 30.csv",
		"watch": "false",
	})
	if err != nil {
		b.Error(err)
	}

	b.Run("GetObservations()", benchGetObservationsFunc(localFileConnector))
}

// Tests "Init()"
func testInitFunc() func(*testing.T) {
	p := NewCsvProcessor()

	params := map[string]string{}

	return func(t *testing.T) {
		err := p.Init(params)
		assert.NoError(t, err)
	}
}

// Tests "GetObservations()"
func testGetObservationsFunc(data []byte) func(*testing.T) {
	return func(t *testing.T) {
		if len(data) == 0 {
			t.Fatal("no data")
		}

		dp := NewCsvProcessor()
		err := dp.Init(nil)
		assert.NoError(t, err)

		_, err = dp.OnData(data)
		assert.NoError(t, err)

		actualObservations, err := dp.GetObservations()
		if err != nil {
			t.Error(err)
			return
		}

		expectedFirstObservation := observations.Observation{
			Time: 1605312000,
			Data: map[string]float64{
				"open":   16339.56,
				"high":   16339.6,
				"low":    16240,
				"close":  16254.51,
				"volume": 274.42607,
			},
		}

		if len(actualObservations[0].Tags) > 0 {
			expectedFirstObservation.Tags = []string{"elon_tweet", "market_open"}
		}

		assert.Equal(t, expectedFirstObservation, actualObservations[0], "First Observation not correct")

		snapshotter.SnapshotT(t, actualObservations)
	}
}

// Tests "GetObservations() - custom time format"
func testGetObservationsCustomTimeFunc() func(*testing.T) {
	return func(t *testing.T) {
		epoch := time.Date(2006, 1, 1, 0, 0, 0, 0, time.UTC)
		period := 7 * 24 * time.Hour
		interval := 24 * time.Hour

		var wg sync.WaitGroup

		localFileConnector := file.NewFileConnector()

		var localData []byte
		err := localFileConnector.Read(func(data []byte, metadata map[string]string) ([]byte, error) {
			localData = data
			wg.Done()
			return nil, nil
		})
		if err != nil {
			t.Fatal(err.Error())
		}
		wg.Add(1)

		err = localFileConnector.Init(epoch, period, interval, map[string]string{
			"path":  "../../test/assets/data/csv/custom_time.csv",
			"watch": "false",
		})
		if err != nil {
			t.Fatal(err.Error())
		}

		if len(localData) == 0 {
			t.Fatal("no data")
		}

		dp := NewCsvProcessor()
		err = dp.Init(map[string]string{
			"time_format": "2006-01-02 15:04:05-07:00",
		})
		assert.NoError(t, err)

		_, err = dp.OnData(localData)
		assert.NoError(t, err)

		actualObservations, err := dp.GetObservations()
		if err != nil {
			t.Error(err)
			return
		}

		expectedFirstObservation := observations.Observation{
			Time: 1547575074,
			Data: map[string]float64{
				"val": 34,
			},
		}
		assert.Equal(t, expectedFirstObservation, actualObservations[0], "First Observation not correct")

		snapshotter.SnapshotT(t, actualObservations)
	}
}

// Tests "GetObservations()" called twice
func testGetObservationsTwiceFunc(data []byte) func(*testing.T) {
	return func(t *testing.T) {
		if len(data) == 0 {
			t.Fatal("no data")
		}

		dp := NewCsvProcessor()
		err := dp.Init(nil)
		assert.NoError(t, err)

		_, err = dp.OnData(data)
		assert.NoError(t, err)

		actualObservations, err := dp.GetObservations()
		assert.NoError(t, err)

		expectedFirstObservation := observations.Observation{
			Time: 1605312000,
			Data: map[string]float64{
				"open":   16339.56,
				"high":   16339.6,
				"low":    16240,
				"close":  16254.51,
				"volume": 274.42607,
			},
		}
		assert.Equal(t, expectedFirstObservation, actualObservations[0], "First Observation not correct")

		actualObservations2, err := dp.GetObservations()
		assert.NoError(t, err)
		assert.Nil(t, actualObservations2)
	}
}

// Tests "GetObservations()" updated with same data
func testGetObservationsSameDataFunc(data []byte) func(*testing.T) {
	return func(t *testing.T) {
		if len(data) == 0 {
			t.Fatal("no data")
		}

		dp := NewCsvProcessor()
		err := dp.Init(nil)
		assert.NoError(t, err)

		_, err = dp.OnData(data)
		assert.NoError(t, err)

		actualObservations, err := dp.GetObservations()
		assert.NoError(t, err)

		expectedFirstObservation := observations.Observation{
			Time: 1605312000,
			Data: map[string]float64{
				"open":   16339.56,
				"high":   16339.6,
				"low":    16240,
				"close":  16254.51,
				"volume": 274.42607,
			},
		}
		assert.Equal(t, expectedFirstObservation, actualObservations[0], "First Observation not correct")

		reader := bytes.NewReader(data)
		buffer := new(bytes.Buffer)
		_, err = io.Copy(buffer, reader)
		if err != nil {
			t.Error(err)
		}

		_, err = dp.OnData(buffer.Bytes())
		assert.NoError(t, err)

		actualObservations2, err := dp.GetObservations()
		assert.NoError(t, err)
		assert.Nil(t, actualObservations2)
	}
}

// Tests "GetState()"
func testGetStateFunc(data []byte) func(*testing.T) {
	return func(t *testing.T) {
		if len(data) == 0 {
			t.Fatal("no data")
		}

		dp := NewCsvProcessor()
		err := dp.Init(nil)
		assert.NoError(t, err)

		_, err = dp.OnData(data)
		assert.NoError(t, err)

		actualState, err := dp.GetState(nil)
		if err != nil {
			t.Error(err)
			return
		}

		assert.Equal(t, 2, len(actualState), "expected two state objects")

		sort.Slice(actualState, func(i, j int) bool {
			return actualState[i].Path() < actualState[j].Path()
		})

		assert.Equal(t, "coinbase.btcusd", actualState[0].Path(), "expected path incorrect")
		assert.Equal(t, "local.portfolio", actualState[1].Path(), "expected path incorrect")

		expectedFirstObservation := observations.Observation{
			Time: 1626697480,
			Data: map[string]float64{
				"price": 31232.709090909084,
			},
		}

		actualObservations := actualState[0].Observations()
		assert.Equal(t, expectedFirstObservation, actualState[0].Observations()[0], "First Observation not correct")
		assert.Equal(t, 57, len(actualObservations), "number of observations incorrect")

		expectedObservations := make([]observations.Observation, 0)
		assert.Equal(t, expectedObservations, actualState[1].Observations(), "Observations not correct")
	}
}

// Tests "GetState()" with tag data
func testGetStateTagsFunc(data []byte) func(*testing.T) {
	return func(t *testing.T) {
		if len(data) == 0 {
			t.Fatal("no data")
		}

		dp := NewCsvProcessor()
		err := dp.Init(nil)
		assert.NoError(t, err)

		_, err = dp.OnData(data)
		assert.NoError(t, err)

		actualState, err := dp.GetState(nil)
		if err != nil {
			t.Error(err)
			return
		}

		assert.Equal(t, 5, len(actualState), "expected two state objects")

		sort.Slice(actualState, func(i, j int) bool {
			return actualState[i].Path() < actualState[j].Path()
		})

		assert.Equal(t, "bitmex.btcusd", actualState[0].Path(), "expected path incorrect")
		assert.Equal(t, "bitthumb.btcusd", actualState[1].Path(), "expected path incorrect")
		assert.Equal(t, "coinbase.btcusd", actualState[2].Path(), "expected path incorrect")
		assert.Equal(t, "coinbase_pro.btcusd", actualState[3].Path(), "expected path incorrect")
		assert.Equal(t, "local.btcusd", actualState[4].Path(), "expected path incorrect")

		expectedFirstObservation := observations.Observation{
			Time: 1605312000,
			Data: map[string]float64{
				"low": 16240,
			},
		}

		actualObservations := actualState[0].Observations()
		assert.Equal(t, expectedFirstObservation, actualState[0].Observations()[0], "First Observation not correct")
		assert.Equal(t, 5, len(actualObservations), "number of observations incorrect")

		testTime := time.Unix(1610057400, 0)
		testTime = testTime.UTC()
		for _, state := range actualState {
			state.Time = testTime
		}

		snapshotter.SnapshotT(t, actualState)
	}
}

// Tests "GetState()" called twice
func testGetStateTwiceFunc(data []byte) func(*testing.T) {
	return func(t *testing.T) {
		if len(data) == 0 {
			t.Fatal("no data")
		}

		dp := NewCsvProcessor()
		err := dp.Init(nil)
		assert.NoError(t, err)

		_, err = dp.OnData(data)
		assert.NoError(t, err)

		actualState, err := dp.GetState(nil)
		if err != nil {
			t.Error(err)
			return
		}

		assert.Equal(t, 2, len(actualState), "expected two state objects")

		sort.Slice(actualState, func(i, j int) bool {
			return actualState[i].Path() < actualState[j].Path()
		})

		assert.Equal(t, "coinbase.btcusd", actualState[0].Path(), "expected path incorrect")
		assert.Equal(t, "local.portfolio", actualState[1].Path(), "expected path incorrect")

		expectedFirstObservation := observations.Observation{
			Time: 1626697480,
			Data: map[string]float64{
				"price": 31232.709090909084,
			},
		}

		actualObservations := actualState[0].Observations()
		assert.Equal(t, expectedFirstObservation, actualState[0].Observations()[0], "First Observation not correct")
		assert.Equal(t, 57, len(actualObservations), "number of observations incorrect")

		expectedObservations := make([]observations.Observation, 0)
		assert.Equal(t, expectedObservations, actualState[1].Observations(), "Observations not correct")

		actualState2, err := dp.GetState(nil)
		assert.NoError(t, err)
		assert.Nil(t, actualState2)
	}
}

// Tests "getColumnMappings()"
func testgetColumnMappingsFunc() func(*testing.T) {
	return func(t *testing.T) {
		headers := []string{"time", "local.portfolio.usd_balance", "local.portfolio.btc_balance", "coinbase.btcusd.price"}

		colToPath, colToFieldName, err := getColumnMappings(headers)
		if err != nil {
			t.Error(err)
			return
		}

		expectedColToPath := []string{"local.portfolio", "local.portfolio", "coinbase.btcusd"}
		assert.Equal(t, expectedColToPath, colToPath, "column to path mapping incorrect")

		expectedColToFieldName := []string{"usd_balance", "btc_balance", "price"}
		assert.Equal(t, expectedColToFieldName, colToFieldName, "column to path mapping incorrect")
	}
}

// Benchmark "GetObservations()"
func benchGetObservationsFunc(c *file.FileConnector) func(*testing.B) {
	return func(b *testing.B) {
		dp := NewCsvProcessor()
		err := dp.Init(nil)
		if err != nil {
			b.Error(err)
		}

		for i := 0; i < 10; i++ {
			_, err := dp.GetObservations()
			if err != nil {
				b.Fatal(err.Error())
			}
		}
	}
}
