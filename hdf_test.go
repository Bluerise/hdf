/*
 * Copyright (c) 2013 Patrick Wildt <patrick@blueri.se>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package hdf_test

import (
	"github.com/Bluerise/hdf"
	"testing"
)

/*
 * Objects with the same name in different trees
 * must not interfere.
 */
func TestHdfFirstChildValue(t *testing.T) {
	obj := hdf.New()
	obj.SetValue("config", "Config Tree")
	obj.SetValue("status", "Status Tree")
	if obj.GetValue("config", "") != "Config Tree" &&
		obj.GetValue("status", "") != "Status Tree" {
		t.Errorf("Failed to set a values.\n")
	}
	obj.SetValue("config.vpn", "VPN in Config Tree")
	obj.SetValue("status.vpn", "VPN in Status Tree")
	if obj.GetValue("config.vpn", "") != "VPN in Config Tree" &&
		obj.GetValue("status.vpn", "") != "VPN in Status Tree" {
		t.Errorf("Failed to set a values.\n")
	}
}

/*
 * An object is allowed to have a value and
 * children at the same time.
 */
func TestHdfObjValueWithChildren(t *testing.T) {
	obj := hdf.New()
	obj.SetValue("config", "Config Tree")
	if obj.GetValue("config", "") != "Config Tree" {
		t.Errorf("Failed to set a simple value.\n")
	}
	obj.SetValue("config.child", "Child")
	if obj.GetValue("config.child", "") != "Child" {
		t.Errorf("Failed to set a value to a child of an object, which has a value.\n")
	}
	if obj.GetValue("config", "") != "Config Tree" {
		t.Errorf("Parent value has been lost/overridden in between.\n")
	}
	obj.SetValue("config", "Config Tree")
	if obj.GetValue("config.child", "") != "Child" {
		t.Errorf("Child value has been lost/overridden in between.\n")
	}
}

func TestHdfLinkedTrees(t *testing.T) {
	obj := hdf.New()
	obj.SetValue("config", "Config Tree")
	obj.SetValue("config.child", "Child")
	obj.LinkValue("link", "config")
	if obj.GetValue("link.child", "") != "Child" {
		t.Errorf("GetValue after Link failed.\n")
	}
	obj.SetValue("link.enable", "1")
	if obj.GetValue("config.enable", "") != "1" {
		t.Errorf("SetValue on Link Child failed.\n")
	}
}
