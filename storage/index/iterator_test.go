// Copyright (c) 2018 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package index

import (
	"fmt"
	"testing"

	"github.com/m3db/m3ninx/doc"
	"github.com/m3db/m3x/ident"
	"github.com/m3db/m3x/resource"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func newTestIterator(i *doc.MockIterator) Iterator {
	return NewIterator(ident.StringID("testNs"), i, NewOptions(), func() {})
}

func newTestIteratorWithFinalizer(i *doc.MockIterator, fn resource.FinalizerFn) Iterator {
	return NewIterator(ident.StringID("testNs"), i, NewOptions(), fn)
}

func TestIteratorEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ri := doc.NewMockIterator(ctrl)
	ri.EXPECT().Next().Return(false)
	ri.EXPECT().Err().Return(nil)

	iter := newTestIterator(ri)
	require.False(t, iter.Next())
}

func TestIteratorEmptyWithFinalizer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var finalizerCalled bool
	fn := func() {
		require.False(t, finalizerCalled)
		finalizerCalled = true
	}
	ri := doc.NewMockIterator(ctrl)
	ri.EXPECT().Next().Return(false)
	ri.EXPECT().Err().Return(nil)

	iter := newTestIteratorWithFinalizer(ri, fn)
	require.False(t, iter.Next())
	require.True(t, finalizerCalled)
}

func TestIteratorWithElements(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var finalizerCalled bool
	fn := func() {
		require.False(t, finalizerCalled)
		finalizerCalled = true
	}

	ri := doc.NewMockIterator(ctrl)
	gomock.InOrder(
		ri.EXPECT().Next().Return(true),
		ri.EXPECT().Current().Return(
			doc.Document{
				Fields: []doc.Field{
					doc.Field{
						Name:  ReservedFieldNameID,
						Value: []byte("foo"),
					},
					doc.Field{
						Name:  []byte("name"),
						Value: []byte("value"),
					},
					doc.Field{
						Name:  []byte("other"),
						Value: []byte("str"),
					},
				},
			},
		),
		ri.EXPECT().Next().Return(false),
		ri.EXPECT().Err().Return(nil),
	)

	iter := newTestIteratorWithFinalizer(ri, fn)
	require.True(t, iter.Next())
	ns, id, tags := iter.Current()
	require.Equal(t, "testNs", ns.String())
	require.Equal(t, "foo", id.String())
	require.Len(t, tags, 2)
	require.Equal(t, "name", tags[0].Name.String())
	require.Equal(t, "value", tags[0].Value.String())
	require.Equal(t, "other", tags[1].Name.String())
	require.Equal(t, "str", tags[1].Value.String())
	require.False(t, iter.Next())
	require.Nil(t, iter.Err())
	require.True(t, finalizerCalled)
}

func TestIteratorWithoutID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var finalizerCalled bool
	fn := func() {
		require.False(t, finalizerCalled)
		finalizerCalled = true
	}

	ri := doc.NewMockIterator(ctrl)
	gomock.InOrder(
		ri.EXPECT().Next().Return(true),
		ri.EXPECT().Current().Return(
			doc.Document{
				Fields: []doc.Field{
					doc.Field{
						Name:  []byte("name"),
						Value: []byte("value"),
					},
					doc.Field{
						Name:  []byte("other"),
						Value: []byte("str"),
					},
				},
			},
		),
	)

	iter := newTestIteratorWithFinalizer(ri, fn)
	require.False(t, iter.Next())
	require.Error(t, iter.Err())
	require.True(t, finalizerCalled)
}

func TestIteratorErr(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var finalizerCalled bool
	fn := func() {
		require.False(t, finalizerCalled)
		finalizerCalled = true
	}

	ri := doc.NewMockIterator(ctrl)
	gomock.InOrder(
		ri.EXPECT().Next().Return(false),
		ri.EXPECT().Err().Return(fmt.Errorf("random-error")),
	)

	iter := newTestIteratorWithFinalizer(ri, fn)
	require.False(t, iter.Next())
	require.NotNil(t, iter.Err())
	require.True(t, finalizerCalled)
}

// TODO(prateek): add a test to ensure we're interacting with ident.Pool as expected