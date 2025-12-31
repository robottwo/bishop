package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSessionOverride_ReturnsValueWhenSet(t *testing.T) {
	// Set up a session override
	sessionConfigOverrides["TEST_KEY"] = "test_value"
	defer delete(sessionConfigOverrides, "TEST_KEY")

	val, ok := GetSessionOverride("TEST_KEY")
	assert.True(t, ok)
	assert.Equal(t, "test_value", val)
}

func TestGetSessionOverride_ReturnsFalseWhenNotSet(t *testing.T) {
	val, ok := GetSessionOverride("NONEXISTENT_KEY")
	assert.False(t, ok)
	assert.Empty(t, val)
}

func TestHomeDir_ReturnsNonEmptyString(t *testing.T) {
	home := homeDir()
	assert.NotEmpty(t, home)
}

func TestMenuItemInterface(t *testing.T) {
	item := menuItem{
		title:       "Test Title",
		description: "Test Description",
	}

	assert.Equal(t, "Test Title", item.Title())
	assert.Equal(t, "Test Description", item.Description())
	assert.Equal(t, "Test Title", item.FilterValue())
}

func TestSettingItemInterface(t *testing.T) {
	item := settingItem{
		title:       "Test Setting",
		description: "Test Description",
		envVar:      "TEST_VAR",
		itemType:    typeText,
	}

	assert.Equal(t, "Test Setting", item.Title())
	assert.Equal(t, "Test Description", item.Description())
	assert.Equal(t, "Test Setting", item.FilterValue())
}

func TestSimpleItemInterface(t *testing.T) {
	item := simpleItem("simple test")

	assert.Equal(t, "simple test", item.Title())
	assert.Empty(t, item.Description())
	assert.Equal(t, "simple test", item.FilterValue())
}

func TestSettingTypes(t *testing.T) {
	assert.Equal(t, settingType(0), typeText)
	assert.Equal(t, settingType(1), typeList)
	assert.Equal(t, settingType(2), typeToggle)
}

func TestStateConstants(t *testing.T) {
	assert.Equal(t, state(0), stateList)
	assert.Equal(t, state(1), stateSubmenu)
	assert.Equal(t, state(2), stateEditing)
	assert.Equal(t, state(3), stateSelection)
}

func TestMenuItemWithSubmenu(t *testing.T) {
	submenuItems := []settingItem{
		{title: "Setting 1", envVar: "VAR1", itemType: typeText},
		{title: "Setting 2", envVar: "VAR2", itemType: typeList},
	}

	item := menuItem{
		title:       "Submenu Item",
		description: "Has a submenu",
		submenu:     submenuItems,
	}

	assert.NotNil(t, item.submenu)
	assert.Len(t, item.submenu, 2)
	assert.Nil(t, item.setting)
}

func TestMenuItemWithDirectSetting(t *testing.T) {
	setting := &settingItem{
		title:    "Direct Setting",
		envVar:   "DIRECT_VAR",
		itemType: typeToggle,
	}

	item := menuItem{
		title:       "Direct Item",
		description: "Has a direct setting",
		setting:     setting,
	}

	assert.Nil(t, item.submenu)
	assert.NotNil(t, item.setting)
	assert.Equal(t, "DIRECT_VAR", item.setting.envVar)
}

func TestSettingItemOptions(t *testing.T) {
	item := settingItem{
		title:    "Provider",
		envVar:   "PROVIDER_VAR",
		itemType: typeList,
		options:  []string{"option1", "option2", "option3"},
	}

	assert.Len(t, item.options, 3)
	assert.Equal(t, "option1", item.options[0])
}
