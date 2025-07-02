package ui

import (
	"context"
	"fmt"
)

// MenuAction represents a function that will be executed when a menu item is selected
type MenuAction func(ctx context.Context) error

// MenuItem represents a menu item with its action
type MenuItem struct {
	Label       string
	Value       string
	Action      MenuAction
	SubMenu     *Menu
	IsEnabled   bool
	Description string
}

// Menu represents a menu with its items and configuration
type Menu struct {
	Title       string
	Items       []MenuItem
	Parent      *Menu
	Type        MenuType
	Description string
}

// MenuManager handles all menu-related operations
type MenuManager struct {
	currentMenu *Menu
	ctx         context.Context
	history     []*Menu
}

// NewMenuManager creates a new menu manager
func NewMenuManager(ctx context.Context) *MenuManager {
	return &MenuManager{
		ctx:     ctx,
		history: make([]*Menu, 0),
	}
}

// NewMenu creates a new menu with the given title and type
func NewMenu(title string, menuType MenuType) *Menu {
	return &Menu{
		Title: title,
		Type:  menuType,
		Items: make([]MenuItem, 0),
	}
}

// AddItem adds a new item to the menu
func (m *Menu) AddItem(label, value string, action MenuAction) *MenuItem {
	item := MenuItem{
		Label:     label,
		Value:     value,
		Action:    action,
		IsEnabled: true,
	}
	m.Items = append(m.Items, item)
	return &m.Items[len(m.Items)-1]
}

// AddSubMenu adds a submenu to a menu item
func (item *MenuItem) AddSubMenu(menu *Menu) *MenuItem {
	item.SubMenu = menu
	menu.Parent = item.SubMenu
	return item
}

// SetDescription sets the description for a menu item
func (item *MenuItem) SetDescription(desc string) *MenuItem {
	item.Description = desc
	return item
}

// SetEnabled sets whether a menu item is enabled
func (item *MenuItem) SetEnabled(enabled bool) *MenuItem {
	item.IsEnabled = enabled
	return item
}

// Show displays the current menu and handles user input
func (mm *MenuManager) Show(menu *Menu) error {
	mm.currentMenu = menu
	mm.history = append(mm.history, menu)

	// Convert menu items to pairs for the existing menu system
	pairs := make([]Pair, 0)
	for _, item := range menu.Items {
		if item.IsEnabled {
			pairs = append(pairs, Pair{
				Label: item.Label,
				Value: item.Value,
			})
		}
	}

	// Show menu using existing menu system
	selected, err := OpenMenu(menu.Type, pairs)
	if err != nil {
		return fmt.Errorf("failed to show menu: %w", err)
	}

	// Find selected item
	var selectedItem *MenuItem
	for i := range menu.Items {
		if menu.Items[i].Value == selected {
			selectedItem = &menu.Items[i]
			break
		}
	}

	if selectedItem == nil {
		return fmt.Errorf("invalid selection: %s", selected)
	}

	// Handle submenu
	if selectedItem.SubMenu != nil {
		return mm.Show(selectedItem.SubMenu)
	}

	// Execute action if present
	if selectedItem.Action != nil {
		return selectedItem.Action(mm.ctx)
	}

	return nil
}

// Back goes back to the previous menu
func (mm *MenuManager) Back() error {
	if len(mm.history) <= 1 {
		return nil
	}

	// Remove current menu from history
	mm.history = mm.history[:len(mm.history)-1]
	// Show previous menu
	return mm.Show(mm.history[len(mm.history)-1])
}

// GetCurrentMenu returns the current menu
func (mm *MenuManager) GetCurrentMenu() *Menu {
	return mm.currentMenu
}

// ClearHistory clears the menu history
func (mm *MenuManager) ClearHistory() {
	mm.history = make([]*Menu, 0)
}
