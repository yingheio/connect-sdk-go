package connect

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/1Password/connect-sdk-go/onepassword"
)

const (
	vaultTag = "opvault"
	itemTag  = "opitem"
	fieldTag = "opfield"

	envVaultVar = "OP_VAULT"
)

type parsedItem struct {
	vaultUUID string
	itemUUID  string
	itemTitle string
	fields    []*reflect.StructField
	values    []*reflect.Value
}

func checkStruct(i interface{}) (reflect.Value, error) {
	configP := reflect.ValueOf(i)
	if configP.Kind() != reflect.Ptr {
		return reflect.Value{}, fmt.Errorf("you must pass a pointer to Config struct")
	}

	config := configP.Elem()
	if config.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("config values can only be loaded into a struct")
	}
	return config, nil

}
func vaultUUIDForField(field *reflect.StructField, vaultUUID string, envVaultFound bool) (string, error) {
	// Check to see if a specific vault has been specified on the field
	// If the env vault id has not been found and item doesn't have a vault
	// return an error
	if vaultUUIDTag := field.Tag.Get(vaultTag); vaultUUIDTag == "" {
		if !envVaultFound {
			return "", fmt.Errorf("There is no vault for %q field", field.Name)
		}
	} else {
		return vaultUUIDTag, nil
	}

	return vaultUUID, nil
}

func setValuesForTag(client Client, parsedItem *parsedItem, byTitle bool) error {
	var item *onepassword.Item
	var err error
	if byTitle {
		item, err = client.GetItemByTitle(parsedItem.itemTitle, parsedItem.vaultUUID)
	} else {
		item, err = client.GetItem(parsedItem.itemUUID, parsedItem.vaultUUID)
	}
	if err != nil {
		return err
	}

	for i, field := range parsedItem.fields {
		value := parsedItem.values[i]
		path := field.Tag.Get(fieldTag)
		if path == "" {
			if field.Type == reflect.TypeOf(onepassword.Item{}) {
				value.Set(reflect.ValueOf(*item))
				return nil
			}
			return fmt.Errorf("There is no %q specified for %q", fieldTag, field.Name)
		}

		pathParts := strings.Split(path, ".")

		if len(pathParts) != 2 {
			return fmt.Errorf("Invalid field path format for %q", field.Name)
		}

		sectionID := sectionIDForName(pathParts[0], item.Sections)
		label := pathParts[1]

		for _, f := range item.Fields {
			fieldSectionID := ""
			if f.Section != nil {
				fieldSectionID = f.Section.ID
			}

			if fieldSectionID == sectionID && f.Label == label {
				if err := setValue(value, f.Value); err != nil {
					return err
				}
				break
			}
		}
	}

	return nil
}

func setValue(value *reflect.Value, toSet string) error {
	switch value.Kind() {
	case reflect.String:
		value.SetString(toSet)
	case reflect.Int:
		v, err := strconv.Atoi(toSet)
		if err != nil {
			return err
		}
		value.SetInt(int64(v))
	default:
		return fmt.Errorf("Unsupported type %q. Only string, int64, and onepassword.Item are supported", value.Kind())
	}

	return nil
}

func sectionIDForName(name string, sections []*onepassword.ItemSection) string {
	if sections == nil {
		return ""
	}

	for _, s := range sections {
		if name == strings.ToLower(s.Label) {
			return s.ID
		}
	}

	return ""
}
