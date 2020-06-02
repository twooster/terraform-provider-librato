package librato

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/heroku/go-librato/librato"
)

func resourceLibratoAlert() *schema.Resource {
	return &schema.Resource{
		Create: resourceLibratoAlertCreate,
		Read:   resourceLibratoAlertRead,
		Update: resourceLibratoAlertUpdate,
		Delete: resourceLibratoAlertDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"active": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"md": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"rearm_seconds": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  600,
			},
			"services": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"condition": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Required: true,
						},
						"metric_name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"source": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"tag": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:     schema.TypeString,
										Required: true,
									},
									"grouped": {
										Type:     schema.TypeBool,
										Optional: true,
										Default:  false,
									},
									"values": {
										Type:     schema.TypeSet,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
								},
							},
						},
						"detect_reset": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"duration": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"threshold": {
							Type:     schema.TypeFloat,
							Optional: true,
						},
						"summary_function": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"attributes": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"runbook_url": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func resourceLibratoAlertCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*librato.Client)

	alert := &librato.Alert{
		Name:   librato.String(d.Get("name").(string)),
		Active: librato.Bool(d.Get("active").(bool)),
		Md:     librato.Bool(d.Get("md").(bool)),
	}

	if v, ok := d.GetOk("description"); ok {
		alert.Description = librato.String(v.(string))
	}
	if v, ok := d.GetOk("rearm_seconds"); ok {
		alert.RearmSeconds = librato.Uint(uint(v.(int)))
	}
	if v, ok := d.GetOk("services"); ok {
		services := []interface{}{}
		for _, serviceData := range v.(*schema.Set).List() {
			serviceID, _ := strconv.Atoi(serviceData.(string))
			services = append(services, serviceID)
		}
		alert.Services = services
	}
	if v, ok := d.GetOk("condition"); ok {
		conditions := make([]librato.AlertCondition, len(v.([]interface{})))
		for i, d := range v.([]interface{}) {
			conditions[i] = expandAlertCondition(d)
		}
		alert.Conditions = conditions
	}
	if v, ok := d.GetOk("attributes"); ok {
		alert.Attributes = expandAlertAttributes(v.([]interface{}))
	}

	log.Printf("[INFO] Creating new alert: %#v", alert)

	alertRes, _, err := client.Alerts.Create(alert)
	if err != nil {
		return fmt.Errorf("Error creating Librato alert %s: %s", *alert.Name, err)
	}

	retryErr := resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, _, err := client.Alerts.Get(*alertRes.ID)
		if err != nil {
			if errResp, ok := err.(*librato.ErrorResponse); ok && errResp.Response.StatusCode == 404 {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if retryErr != nil {
		return fmt.Errorf("Error creating librato alert: %s", err)
	}

	d.SetId(strconv.FormatUint(uint64(*alertRes.ID), 10))
	return resourceLibratoAlertRead(d, meta)
}

func resourceLibratoAlertRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*librato.Client)

	id, err := strconv.ParseUint(d.Id(), 10, 0)
	if err != nil {
		return err
	}

	alert, _, err := client.Alerts.Get(uint(id))

	if err != nil {
		if errResp, ok := err.(*librato.ErrorResponse); ok && errResp.Response.StatusCode == 404 {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error reading Librato Alert %s: %s", d.Id(), err)
	}

	log.Printf("[INFO] Librato alert read: %#v", alert)

	if err := d.Set("name", *alert.Name); err != nil {
		return err
	}
	if alert.Description != nil {
		if err := d.Set("description", *alert.Description); err != nil {
			return err
		}
	}
	if alert.RearmSeconds != nil && *alert.RearmSeconds != 600 {
		if err := d.Set("rearm_seconds", *alert.RearmSeconds); err != nil {
			return err
		}
	}
	services := make([]interface{}, 0, len(alert.Services.([]interface{})))
	for _, s := range alert.Services.([]interface{}) {
		data := s.(map[string]interface{})
		services = append(services, fmt.Sprintf("%.f", data["id"]))
	}
	if err := d.Set("services", schema.NewSet(schema.HashString, services)); err != nil {
		return err
	}
	if len(alert.Conditions) > 0 {
		conditions := make([]interface{}, len(alert.Conditions))
		for i, cnd := range alert.Conditions {
			conditions[i] = flattenCondition(cnd)
		}
		if err := d.Set("condition", conditions); err != nil {
			return err
		}
	}
	if alert.Attributes != nil {
		if err := d.Set("attributes", flattenAttributes(alert.Attributes)); err != nil {
			return err
		}
	}
	return nil
}

func resourceLibratoAlertUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*librato.Client)

	id, err := strconv.ParseUint(d.Id(), 10, 0)
	if err != nil {
		return err
	}

	alert := librato.Alert{
		Name:   librato.String(d.Get("name").(string)),
		Active: librato.Bool(d.Get("active").(bool)),
		Md:     librato.Bool(d.Get("md").(bool)),
	}

	if d.HasChange("description") {
		alert.Description = librato.String(d.Get("description").(string))
	}
	if d.HasChange("rearm_seconds") {
		alert.RearmSeconds = librato.Uint(uint(d.Get("rearm_seconds").(int)))
	}
	if d.HasChange("services") {
		vs := d.Get("services").(*schema.Set).List()
		services := make([]*int, len(vs))
		for i, d := range vs {
			serviceID, err := strconv.Atoi(d.(string))
			if err == nil {
				services[i] = librato.Int(serviceID)
			}
		}
		alert.Services = services
	}
	if d.HasChange("condition") {
		vs := d.Get("condition").([]interface{})
		conditions := make([]librato.AlertCondition, len(vs))
		for i, d := range vs {
			conditions[i] = expandAlertCondition(d)
		}
		alert.Conditions = conditions
	}
	if d.HasChange("attributes") {
		v, ok := d.GetOk("attributes")

		// If no attributes are defined, just set to empty attributes.
		if !ok {
			alert.Attributes = &librato.AlertAttributes{
				RunbookURL: librato.String(""),
			}
		} else {
			attributeData := v.([]interface{})
			if attributeData[0] == nil {
				return fmt.Errorf("No attributes found in attributes block")
			}
			attributeDataMap := attributeData[0].(map[string]interface{})
			attributes := new(librato.AlertAttributes)
			if v, ok := attributeDataMap["runbook_url"].(string); ok && v != "" {
				attributes.RunbookURL = librato.String(v)
			}
			alert.Attributes = attributes
		}
	}

	log.Printf("[INFO] Updating Librato alert: %#v", alert)

	_, updErr := client.Alerts.Update(uint(id), &alert)
	if updErr != nil {
		return fmt.Errorf("Error updating Librato alert: %s", updErr)
	}

	log.Printf("[INFO] Updated Librato alert %#v", alert)

	// Wait for propagation since Librato updates are eventually consistent
	wait := resource.StateChangeConf{
		Pending:                   []string{fmt.Sprintf("%t", false)},
		Target:                    []string{fmt.Sprintf("%t", true)},
		Timeout:                   5 * time.Minute,
		MinTimeout:                2 * time.Second,
		ContinuousTargetOccurence: 5,
		Refresh: func() (interface{}, string, error) {
			log.Printf("[DEBUG] Checking if Librato Alert %d was updated yet", id)

			changedAlert, _, getErr := client.Alerts.Get(uint(id))
			if getErr != nil {
				return changedAlert, "", getErr
			}

			log.Printf("[INFO] Updated alert: %#v", changedAlert)
			return changedAlert, "true", nil
		},
	}

	_, err = wait.WaitForState()
	if err != nil {
		return fmt.Errorf("Failed updating Librato Alert %d: %s", id, err)
	}

	return resourceLibratoAlertRead(d, meta)
}

func resourceLibratoAlertDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*librato.Client)

	id, err := strconv.ParseUint(d.Id(), 10, 0)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Deleting Alert: %d", id)

	_, err = client.Alerts.Delete(uint(id))
	if err != nil {
		return fmt.Errorf("Error deleting Alert: %s", err)
	}

	retryErr := resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, _, err := client.Alerts.Get(uint(id))
		if err != nil {
			if errResp, ok := err.(*librato.ErrorResponse); ok && errResp.Response.StatusCode == 404 {
				return nil
			}
			return resource.NonRetryableError(err)
		}
		return resource.RetryableError(fmt.Errorf("alert still exists"))
	})
	if retryErr != nil {
		return fmt.Errorf("Error deleting librato alert: %s", err)
	}

	return nil
}

// Expanders

func expandAlertCondition(in interface{}) librato.AlertCondition {
	condition := librato.AlertCondition{}
	m := in.(map[string]interface{})
	if v, ok := m["type"].(string); ok {
		condition.Type = librato.String(v)
	}
	if v, ok := m["metric_name"].(string); ok && len(v) > 0 {
		condition.MetricName = librato.String(v)
	}
	if v, ok := m["source"].(string); ok && len(v) > 0 {
		condition.Source = librato.String(v)
	}
	if v, ok := m["detect_reset"].(bool); ok {
		condition.DetectReset = librato.Bool(v)
	}
	if v, ok := m["duration"].(int); ok {
		condition.Duration = librato.Uint(uint(v))
	}
	if v, ok := m["summary_function"].(string); ok {
		condition.SummaryFunction = librato.String(v)
	}
	if v, ok := m["threshold"].(float64); ok && !math.IsNaN(v) {
		condition.Threshold = librato.Float(v)
	}
	if v, ok := m["tag"].([]interface{}); ok && len(v) > 0 {
		tags := make([]librato.AlertConditionTagSet, len(v))
		for i, t := range v {
			tags[i] = expandAlertConditionTagSet(t)
		}
		condition.Tags = tags
	}
	return condition
}

func expandAlertConditionTagSet(in interface{}) librato.AlertConditionTagSet {
	tag := librato.AlertConditionTagSet{}
	m := in.(map[string]interface{})
	if v, ok := m["name"].(string); ok {
		tag.Name = librato.String(v)
	}
	if v, ok := m["grouped"].(bool); ok {
		tag.Grouped = librato.Bool(v)
	}
	if v, ok := m["values"]; ok {
		setList := v.(*schema.Set).List()
		values := make([]*string, len(setList))
		for i, value := range setList {
			values[i] = librato.String(value.(string))
		}
		tag.Values = values
	}
	return tag
}

func expandAlertAttributes(in []interface{}) *librato.AlertAttributes {
	if len(in) == 0 || in[0] == nil {
		return &librato.AlertAttributes{}
	}

	attr := &librato.AlertAttributes{}
	m := in[0].(map[string]interface{})
	if v, ok := m["runbook_url"].(string); ok {
		attr.RunbookURL = librato.String(v)
	}
	return attr
}

// Flatteners

func flattenCondition(condition librato.AlertCondition) interface{} {
	m := make(map[string]interface{}, 0)
	if condition.Type != nil {
		m["type"] = *condition.Type
	}
	if condition.MetricName != nil {
		m["metric_name"] = *condition.MetricName
	}
	if condition.Source != nil {
		m["source"] = *condition.Source
	}
	if condition.DetectReset != nil {
		m["detect_reset"] = *condition.DetectReset
	}
	if condition.Threshold != nil {
		m["threshold"] = *condition.Threshold
	}
	if condition.SummaryFunction != nil {
		m["summary_function"] = *condition.SummaryFunction
	}
	if condition.Duration != nil {
		m["duration"] = *condition.Duration
	}
	if len(condition.Tags) > 0 {
		flattenedTags := make([]interface{}, len(condition.Tags))
		for i, tag := range condition.Tags {
			flattenedTags[i] = flattenTagsReferance(tag)
		}
		m["tag"] = flattenedTags
	}
	return m
}

func flattenTagsReferance(tag librato.AlertConditionTagSet) interface{} {
	m := make(map[string]interface{}, 0)
	if tag.Name != nil {
		m["name"] = *tag.Name
	}
	if tag.Grouped != nil {
		m["grouped"] = *tag.Grouped
	}
	if len(tag.Values) > 0 {
		values := make([]interface{}, len(tag.Values))
		for i, value := range tag.Values {
			values[i] = *value
		}
		m["values"] = schema.NewSet(schema.HashString, values)
	}
	return m
}

func flattenAttributes(attr *librato.AlertAttributes) []interface{} {
	m := make(map[string]interface{}, 0)
	if attr.RunbookURL != nil {
		m["runbook_url"] = *attr.RunbookURL
	}
	return []interface{}{m}
}
