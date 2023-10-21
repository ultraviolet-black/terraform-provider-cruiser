terraform {
  required_providers {
    cruiser = {
      source = "ultraviolet-black/cruiser"
    }
  }
}

provider "cruiser" {}

resource "cruiser_envoy_cluster_load_assignment" "main" {
  name = "my-cluster"
  cluster_name = "my-cluster"
}