/* Copyright (c) 2012 The ANTLR Project Contributors. All rights reserved.
 * Use is of this file is governed by the BSD 3-clause license that
 * can be found in the LICENSE.txt file in the project root.
 */
package antlr

type IntStream interface {
	Consume()
	LA(int) int
	Mark() int
	Release(marker int)
	Index() int
	Seek(index int)
	Size() int
	GetSourceName() string
}
