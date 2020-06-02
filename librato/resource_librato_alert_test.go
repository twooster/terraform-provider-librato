package librato

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/heroku/go-librato/librato"
)

func TestAccLibratoAlert_Minimal(t *testing.T) {
	var alert librato.Alert
	name := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLibratoAlertDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckLibratoAlertConfig_minimal(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLibratoAlertExists("librato_alert.foobar", &alert),
					testAccCheckLibratoAlertName(&alert, name),
					resource.TestCheckResourceAttr("librato_alert.foobar", "name", name),
				),
			},
		},
	})
}

func TestAccLibratoAlert_Basic(t *testing.T) {
	var alert librato.Alert
	name := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLibratoAlertDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckLibratoAlertConfig_basic(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLibratoAlertExists("librato_alert.foobar", &alert),
					testAccCheckLibratoAlertName(&alert, name),
					testAccCheckLibratoAlertDescription(&alert, "A Test Alert"),
					resource.TestCheckResourceAttr(
						"librato_alert.foobar", "name", name),
				),
			},
		},
	})
}

func TestAccLibratoAlert_FullCreate(t *testing.T) {
	var alert librato.Alert
	prefix := "test"
	name := fmt.Sprintf("%s-%s", prefix, acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLibratoAlertDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckLibratoAlertConfig_full(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLibratoAlertExists("librato_alert.foobar", &alert),
					testAccCheckLibratoAlertName(&alert, name),
					testAccCheckLibratoAlertDescription(&alert, "A Test Alert"),
					resource.TestCheckResourceAttr("librato_alert.foobar", "name", name),
					testAccCheckLibratoAlert(&alert),
				),
			},
		},
	})
}

func TestAccLibratoAlert_Updated(t *testing.T) {
	var alert librato.Alert
	name := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLibratoAlertDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckLibratoAlertConfig_basic(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLibratoAlertExists("librato_alert.foobar", &alert),
					testAccCheckLibratoAlertDescription(&alert, "A Test Alert"),
					resource.TestCheckResourceAttr(
						"librato_alert.foobar", "name", name),
				),
			},
			{
				Config: testAccCheckLibratoAlertConfig_new_value(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLibratoAlertExists("librato_alert.foobar", &alert),
					testAccCheckLibratoAlertDescription(&alert, "A modified Test Alert"),
					resource.TestCheckResourceAttr(
						"librato_alert.foobar", "description", "A modified Test Alert"),
				),
			},
		},
	})
}

func TestAccLibratoAlert_Rename(t *testing.T) {
	var alert librato.Alert
	name := acctest.RandString(10)
	newName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLibratoAlertDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckLibratoAlertConfig_basic(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLibratoAlertExists("librato_alert.foobar", &alert),
					resource.TestCheckResourceAttr(
						"librato_alert.foobar", "name", name),
				),
			},
			{
				Config: testAccCheckLibratoAlertConfig_basic(newName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLibratoAlertExists("librato_alert.foobar", &alert),
					resource.TestCheckResourceAttr(
						"librato_alert.foobar", "name", newName),
				),
			},
		},
	})
}

func TestAccLibratoAlert_FullUpdate(t *testing.T) {
	var alert librato.Alert
	name := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLibratoAlertDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckLibratoAlertConfig_full_update(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLibratoAlertExists("librato_alert.foobar", &alert),
					testAccCheckLibratoAlertName(&alert, name),
					testAccCheckLibratoAlertDescription(&alert, "A Test Alert Updated"),
					testAccCheckLibratoAlertUpdate(&alert),
				),
			},
		},
	})
}

func testAccCheckLibratoAlertDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*librato.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "librato_alert" {
			continue
		}

		id, err := strconv.ParseUint(rs.Primary.ID, 10, 0)
		if err != nil {
			return fmt.Errorf("ID not a number")
		}

		_, _, err = client.Alerts.Get(uint(id))

		if err == nil {
			return fmt.Errorf("Alert still exists")
		}
	}

	return nil
}

func testAccCheckLibratoAlertName(alert *librato.Alert, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if alert.Name == nil || *alert.Name != name {
			return fmt.Errorf("Bad name: %s", *alert.Name)
		}

		return nil
	}
}

func testAccCheckLibratoAlertDescription(alert *librato.Alert, description string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if alert.Description == nil || *alert.Description != description {
			return fmt.Errorf("Bad description: %s", *alert.Description)
		}

		return nil
	}
}

func testAccCheckLibratoAlert(alert *librato.Alert) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if alert.ID == nil {
			return errors.New("Bad alert.ID = nil")
		}

		if *alert.Active == true {
			return fmt.Errorf("Bad alert.active: %t", *alert.Active)
		}

		if alert.RearmSeconds == nil || *alert.RearmSeconds != 300 {
			return fmt.Errorf("Bad alert.rearm_seconds: %d", *alert.RearmSeconds)
		}

		// services
		services := alert.Services.([]interface{})
		if len(services) != 1 {
			return fmt.Errorf("Bad alert.services: len(%d)", len(services))
		}

		service := services[0].(map[string]interface{})
		if title, ok := service["title"].(string); !ok || title != "Foo Bar" {
			return fmt.Errorf("Bad alert.services.title: %s", title)
		}

		if typ, ok := service["type"].(string); !ok || typ != "mail" {
			return fmt.Errorf("Bad alert.services.type: %s", typ)
		}

		if settings, ok := service["settings"].(map[string]interface{}); !ok {
			if addresses, ok := settings["addresses"].(string); !ok || addresses != "admin@example.com" {
				return fmt.Errorf("Bad alert.services.settings.addresses: %s", addresses)
			}
		}

		// conditions
		if len(alert.Conditions) != 1 {
			return fmt.Errorf("Bad conditions: len(%d)", len(alert.Conditions))
		}

		condition := alert.Conditions[0]

		if *condition.Type != "above" {
			return fmt.Errorf("Bad condition.type: %s", *condition.Type)
		}

		if *condition.Threshold != 10 {
			return fmt.Errorf("Bad condition.threshold: %f", *condition.Threshold)
		}

		if *condition.Duration != 600 {
			return fmt.Errorf("Bad condition.duration: %d", *condition.Duration)
		}

		if *condition.MetricName != "librato.cpu.percent.idle" {
			return fmt.Errorf("Bad condition.metric_name: %s", *condition.MetricName)
		}

		if len(condition.Tags) != 2 {
			return fmt.Errorf("Bad condition.tags: len(%d)", len(condition.Tags))
		}

		// condition.tags
		for _, tag := range condition.Tags {
			if tag.Name == nil || (*tag.Name != "tagname" && *tag.Name != "tagname2") {
				return fmt.Errorf("Bad condition.tags: %s", *tag.Name)
			}

			if *tag.Name == "tagname" {
				if tag.Grouped == nil || *tag.Grouped == true {
					return fmt.Errorf("Bad condition.tags.grouped: %t", *tag.Grouped)
				}

				if len(tag.Values) != 2 {
					return fmt.Errorf("Bad condition.tags.values: len(%d)", len(tag.Values))
				}
			}

			if *tag.Name == "tagname2" {
				if tag.Grouped == nil || *tag.Grouped == false {
					return fmt.Errorf("Bad condition.tags.grouped: %t", *tag.Grouped)
				}
			}
		}

		// attributes
		if alert.Attributes.RunbookURL == nil || *alert.Attributes.RunbookURL != "https://www.youtube.com/watch?v=oHg5SJYRHA0" {
			return fmt.Errorf("Bad attributes.runbook_url: %s", *alert.Attributes.RunbookURL)
		}

		return nil
	}
}

func testAccCheckLibratoAlertUpdate(alert *librato.Alert) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if alert.ID == nil {
			return errors.New("Bad alert.ID = nil")
		}

		if *alert.Active == true {
			return fmt.Errorf("Bad alert.active: %t", *alert.Active)
		}

		if alert.RearmSeconds == nil || *alert.RearmSeconds != 1200 {
			return fmt.Errorf("Bad alert.rearm_seconds: %d", *alert.RearmSeconds)
		}

		// services
		services := alert.Services.([]interface{})
		if len(services) != 1 {
			return fmt.Errorf("Bad alert.services: len(%d)", len(services))
		}

		service := services[0].(map[string]interface{})
		if title, ok := service["title"].(string); !ok || title != "Foo Bar" {
			return fmt.Errorf("Bad alert.services.title: %s", title)
		}

		if typ, ok := service["type"].(string); !ok || typ != "mail" {
			return fmt.Errorf("Bad alert.services.type: %s", typ)
		}

		if settings, ok := service["settings"].(map[string]interface{}); !ok {
			if addresses, ok := settings["addresses"].(string); !ok || addresses != "admin@example.com" {
				return fmt.Errorf("Bad alert.services.settings.addresses: %s", addresses)
			}
		}

		// conditions
		if len(alert.Conditions) != 1 {
			return fmt.Errorf("Bad conditions: len(%d)", len(alert.Conditions))
		}

		condition := alert.Conditions[0]

		if *condition.Type != "above" {
			return fmt.Errorf("Bad condition.type: %s", *condition.Type)
		}

		if *condition.Threshold != 9 {
			return fmt.Errorf("Bad condition.threshold: %f", *condition.Threshold)
		}

		if *condition.Duration != 60 {
			return fmt.Errorf("Bad condition.duration: %d", *condition.Duration)
		}

		if *condition.MetricName != "librato.cpu.percent.idle" {
			return fmt.Errorf("Bad condition.metric_name: %s", *condition.MetricName)
		}

		if len(condition.Tags) != 2 {
			return fmt.Errorf("Bad condition.tags: len(%d)", len(condition.Tags))
		}

		// condition.tags
		for _, tag := range condition.Tags {
			if tag.Name == nil || (*tag.Name != "tagname-updated" && *tag.Name != "tagname2-updated") {
				return fmt.Errorf("Bad condition.tags: %s", *tag.Name)
			}

			if *tag.Name == "tagname-updated" {
				if tag.Grouped == nil || *tag.Grouped == false {
					return fmt.Errorf("Bad condition.tags.grouped: %t", *tag.Grouped)
				}
			}

			if *tag.Name == "tagname2-updated" {
				if tag.Grouped == nil || *tag.Grouped == true {
					return fmt.Errorf("Bad condition.tags.grouped: %t", *tag.Grouped)
				}

				if len(tag.Values) != 2 {
					return fmt.Errorf("Bad condition.tags.values: len(%d)", len(tag.Values))
				}
			}
		}

		// attributes
		if alert.Attributes.RunbookURL == nil || *alert.Attributes.RunbookURL != "https://www.youtube.com/watch?v=oHg5SJYRHA0+updated" {
			return fmt.Errorf("Bad attributes.runbook_url: %s", *alert.Attributes.RunbookURL)
		}

		return nil
	}
}

func testAccCheckLibratoAlertExists(n string, alert *librato.Alert) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Alert ID is set")
		}

		client := testAccProvider.Meta().(*librato.Client)

		id, err := strconv.ParseUint(rs.Primary.ID, 10, 0)
		if err != nil {
			return fmt.Errorf("ID not a number")
		}

		foundAlert, _, err := client.Alerts.Get(uint(id))

		if err != nil {
			return err
		}

		if foundAlert.ID == nil || *foundAlert.ID != uint(id) {
			return fmt.Errorf("Alert not found")
		}

		*alert = *foundAlert

		return nil
	}
}

func testAccCheckLibratoAlertConfig_minimal(name string) string {
	return fmt.Sprintf(`
resource "librato_alert" "foobar" {
    name = "%s"
}`, name)
}

func testAccCheckLibratoAlertConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "librato_alert" "foobar" {
    name = "%s"
    description = "A Test Alert"
}`, name)
}

func testAccCheckLibratoAlertConfig_new_value(name string) string {
	return fmt.Sprintf(`
resource "librato_alert" "foobar" {
    name = "%s"
    description = "A modified Test Alert"
}`, name)
}

func testAccCheckLibratoAlertConfig_full(name string) string {
	return fmt.Sprintf(`
resource "librato_service" "foobar" {
    title = "Foo Bar"
    type = "mail"
    settings = <<EOF
{
  "addresses": "admin@example.com"
}
EOF
}

resource "librato_alert" "foobar" {
    name = "%s"
    description = "A Test Alert"
    services = [ "${librato_service.foobar.id}" ]

    condition {
      type = "above"
      threshold = 10
      duration = 600
      metric_name = "librato.cpu.percent.idle"

	  tag {
		name = "tagname"
		grouped = false
		values = [ "value1", "value2" ]
	  }

	  tag {
		name = "tagname2"
		grouped = true
	  }
    }

    attributes {
      runbook_url = "https://www.youtube.com/watch?v=oHg5SJYRHA0"
    }
    active = false
    rearm_seconds = 300
}`, name)
}

func testAccCheckLibratoAlertConfig_full_update(name string) string {
	return fmt.Sprintf(`
resource "librato_service" "foobar" {
    title = "Foo Bar"
    type = "mail"
    settings = <<EOF
{
  "addresses": "admin@example.com"
}
EOF
}

resource "librato_alert" "foobar" {
    name = "%s"
    description = "A Test Alert Updated"
    services = [ "${librato_service.foobar.id}" ]

    condition {
      type = "above"
      threshold = 9
      duration = 60
      metric_name = "librato.cpu.percent.idle"

	  tag {
		name = "tagname-updated"
		grouped = true
	  }

	  tag {
		name = "tagname2-updated"
		grouped = false
		values = [ "value3-updated", "value4-updated" ]
	  }
    }

    attributes {
      runbook_url = "https://www.youtube.com/watch?v=oHg5SJYRHA0+updated"
    }

    active = false
    rearm_seconds = 1200
}`, name)
}
