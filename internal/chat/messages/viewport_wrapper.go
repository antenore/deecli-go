// Copyright 2025 Antenore Gatta
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package messages

import "github.com/charmbracelet/bubbles/viewport"

// ViewportWrapper wraps the bubbletea viewport to implement ViewportInterface
type ViewportWrapper struct {
	*viewport.Model
}

// NewViewportWrapper creates a new viewport wrapper
func NewViewportWrapper(vp *viewport.Model) *ViewportWrapper {
	return &ViewportWrapper{Model: vp}
}

// GetWidth returns the viewport width
func (vw *ViewportWrapper) GetWidth() int {
	return vw.Width
}

// GotoBottom scrolls to bottom (override to match interface)
func (vw *ViewportWrapper) GotoBottom() {
	vw.Model.GotoBottom()
}