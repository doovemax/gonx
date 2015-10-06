package gonx

import (
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestFilter(t *testing.T) {
	Convey("Test Datetime filter", t, func() {
		start := time.Date(2015, time.February, 2, 2, 2, 2, 0, time.UTC)
		end := time.Date(2015, time.May, 5, 5, 5, 5, 0, time.UTC)

		jan := NewEntry(Fields{"timestamp": "2015-01-01T01:01:01Z", "foo": "12"})
		feb := NewEntry(Fields{"timestamp": "2015-02-02T02:02:02Z", "foo": "34"})
		mar := NewEntry(Fields{"timestamp": "2015-03-03T03:03:03Z", "foo": "56"})
		apr := NewEntry(Fields{"timestamp": "2015-04-04T04:04:04Z", "foo": "78"})
		may := NewEntry(Fields{"timestamp": "2015-05-05T05:05:05Z", "foo": "90"})

		Convey("Filter Entry", func() {
			Convey("Start and end", func() {
				filter := &Datetime{
					Field:  "timestamp",
					Format: time.RFC3339,
					Start:  start,
					End:    end,
				}

				// entries is out of datetime range
				So(filter.Filter(jan), ShouldBeNil)
				So(filter.Filter(may), ShouldBeNil)

				// entry's timestamp meets filter condition
				So(filter.Filter(feb), ShouldEqual, feb)
			})

			Convey("Start only", func() {
				filter := &Datetime{
					Field:  "timestamp",
					Format: time.RFC3339,
					Start:  start,
				}

				// entry is out of datetime range
				So(filter.Filter(jan), ShouldBeNil)

				// entry's timestamp meets filter condition
				So(filter.Filter(feb), ShouldEqual, feb)
			})

			Convey("End only", func() {
				filter := &Datetime{
					Field:  "timestamp",
					Format: time.RFC3339,
					End:    end,
				}

				// entry's timestamp meets filter condition
				So(filter.Filter(jan), ShouldEqual, jan)

				// entry is out of datetime range
				So(filter.Filter(may), ShouldBeNil)
			})
		})

		Convey("Reduce channel", func() {
			filter := &Datetime{
				Field:  "timestamp",
				Format: time.RFC3339,
				Start:  start,
				End:    end,
			}

			// Prepare input channel
			input := make(chan *Entry, 5)
			input <- jan
			input <- feb
			input <- mar
			input <- apr
			input <- may
			close(input)

			output := make(chan *Entry, 5) // Make it buffered to avoid deadlock
			filter.Reduce(input, output)

			expected := []string{
				"'timestamp'=2015-02-02T02:02:02Z;'foo'=34",
				"'timestamp'=2015-03-03T03:03:03Z;'foo'=56",
				"'timestamp'=2015-04-04T04:04:04Z;'foo'=78",
			}
			results := []string{}

			for result := range output {
				results = append(
					results,
					result.FieldsHash([]string{"timestamp", "foo"}),
				)
			}
			So(results, ShouldResemble, expected)
		})
	})
}

func TestChainFilterWithRedicer(t *testing.T) {
	// Prepare input channel
	input := make(chan *Entry, 5)
	input <- NewEntry(Fields{
		"timestamp": "2015-01-01T01:01:01Z",
		"foo":       "12",
		"bar":       "34",
		"baz":       "56",
	})
	input <- NewEntry(Fields{
		"timestamp": "2015-02-02T02:02:02Z",
		"foo":       "34",
		"bar":       "56",
		"baz":       "78",
	})
	input <- NewEntry(Fields{
		"timestamp": "2015-04-04T04:04:04Z",
		"foo":       "78",
		"bar":       "90",
		"baz":       "12",
	})
	input <- NewEntry(Fields{
		"timestamp": "2015-05-05T05:05:05Z",
		"foo":       "90",
		"bar":       "34",
		"baz":       "56",
	})
	close(input)

	filter := &Datetime{
		Field:  "timestamp",
		Format: time.RFC3339,
		Start:  time.Date(2015, time.February, 2, 2, 2, 2, 0, time.UTC),
		End:    time.Date(2015, time.May, 5, 5, 5, 5, 0, time.UTC),
	}
	chain := NewChain(filter, &Avg{[]string{"foo", "bar"}}, &Count{})

	output := make(chan *Entry, 5) // Make it buffered to avoid deadlock
	chain.Reduce(input, output)

	result, ok := <-output
	assert.True(t, ok)

	value, err := result.FloatField("foo")
	assert.NoError(t, err)
	assert.Equal(t, value, (34.0+78)/2.0)

	value, err = result.FloatField("bar")
	assert.NoError(t, err)
	assert.Equal(t, value, (56.0+90)/2.0)

	count, err := result.Field("count")
	assert.NoError(t, err)
	assert.Equal(t, count, "2")

	_, err = result.Field("buz")
	assert.Error(t, err)
}
