---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cruiser_envoy_route_configuration Resource - cruiser"
subcategory: ""
description: |-
  Envoy RouteConfiguration Resource
---

# cruiser_envoy_route_configuration (Resource)

Envoy RouteConfiguration Resource



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) The name of the Envoy RouteConfiguration Resource
- `vhds` (Attributes) VHDS (see [below for nested schema](#nestedatt--vhds))

### Optional

- `ignore_port_in_host_matching` (Boolean) Ignore port in host matching

### Read-Only

- `proto_json` (String) The Envoy RouteConfiguration Resource in JSON format

<a id="nestedatt--vhds"></a>
### Nested Schema for `vhds`

Required:

- `api_type` (String) The api type of the route specifier
- `envoy_grpc_cluster_name` (String) The envoy grpc cluster name of the route specifier
- `initial_fetch_timeout` (String) Initial Fetch Timeout

Optional:

- `resource_api_version` (String) The resource api version of the route specifier
- `set_node_on_first_message_only` (Boolean) The set node on first message only of the route specifier
- `transport_api_version` (String) The transport api version of the route specifier
