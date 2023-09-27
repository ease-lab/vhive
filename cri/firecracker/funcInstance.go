// MIT License
//
// Copyright (c) 2023 Georgiy Lebedev, Plamen Petrov and vHive team
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package firecracker

import (
	log "github.com/sirupsen/logrus"
	"github.com/vhive-serverless/vhive/ctriface"
)

type funcInstance struct {
	VmID            string
	Image           string
	Revision        string
	Logger          *log.Entry
	SnapBooted      bool
	StartVMResponse *ctriface.StartVMResponse
}

func newFuncInstance(vmID, image, revision string, snapBooted bool, startVMResponse *ctriface.StartVMResponse) *funcInstance {
	f := &funcInstance{
		VmID:            vmID,
		Image:           image,
		Revision:        revision,
		SnapBooted:      snapBooted,
		StartVMResponse: startVMResponse,
	}

	f.Logger = log.WithFields(
		log.Fields{
			"vmID":     vmID,
			"image":    image,
			"revision": revision,
		},
	)

	return f
}
