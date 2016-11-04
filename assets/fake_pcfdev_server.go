package main

import (
	"net/http"
	"encoding/json"
	"log"
)

const BINDING_GUID = "pcfdev"

func main() {
	http.HandleFunc("/v2/service_plans", func(w http.ResponseWriter, r *http.Request) {
		if (r.Method == "GET") {
			w.Write([]byte(`
{
   "total_results": 1,
   "total_pages": 1,
   "resources": [
      {
         "metadata": {
            "guid": "40d4bcfe-fccd-4b0a-9e41-e518f9fe4001",
            "url": "/v2/service_plans/40d4bcfe-fccd-4b0a-9e41-e518f9fe4001",
         },
         "entity": {
            "service_guid": "6544d67e-ea8c-42b4-a528-5953d0a0552f",
            "unique_id": "ab08f1bc-e6fc-4b56-a767-ee0fea6e3f20",
            "service_url": "/v2/services/6544d67e-ea8c-42b4-a528-5953d0a0552f",
            "service_instances_url": "/v2/service_plans/40d4bcfe-fccd-4b0a-9e41-e518f9fe4001/service_instances"
         }
      }
   ]
}
			`))
		}
	})

	http.HandleFunc("/v2/services/6544d67e-ea8c-42b4-a528-5953d0a0552f", func(w http.ResponseWriter, r *http.Request) {
		if (r.Method == "GET") {
			w.Write([]byte(`
{
   "metadata": {
      "guid": "6544d67e-ea8c-42b4-a528-5953d0a0552f",
      "url": "/v2/services/6544d67e-ea8c-42b4-a528-5953d0a0552f",
   },
   "entity": {
      "unique_id": "44b26033-1f54-4087-b7bc-da9652c2a539",
      "tags": [
         "mysql"
      ],
      "service_broker_guid": "0761af0d-4d5a-4aab-a288-39b40a3a09d7",
      "service_plans_url": "/v2/services/6544d67e-ea8c-42b4-a528-5953d0a0552f/service_plans"
   }
}
			`))
		}
	})

	http.HandleFunc("/v2/service_plans/40d4bcfe-fccd-4b0a-9e41-e518f9fe4001/service_instances", func(w http.ResponseWriter, r *http.Request) {
		if (r.Method == "GET") {
			w.Write([]byte(`
{
   "total_results": 1,
   "total_pages": 1,
   "resources": [
      {
         "metadata": {
            "guid": "7932fe33-e3ac-4cff-9e08-980313230a89",
            "url": "/v2/service_instances/7932fe33-e3ac-4cff-9e08-980313230a89",
         },
         "entity": {
		"name": "my-mysql-2",
            "service_plan_guid": "40d4bcfe-fccd-4b0a-9e41-e518f9fe4001",
            "space_guid": "4ae5d97e-d1c8-49a2-8c73-466bc307f81f",
            "dashboard_url": "http://mysql-broker.local.pcfdev.io/manage/instances/7932fe33-e3ac-4cff-9e08-980313230a89",
            "space_url": "/v2/spaces/4ae5d97e-d1c8-49a2-8c73-466bc307f81f"
         }
      }
   ]
}			`))
		}
	})

	http.HandleFunc("/v2/spaces/4ae5d97e-d1c8-49a2-8c73-466bc307f81f", func(w http.ResponseWriter, r *http.Request) {
		if (r.Method == "GET") {
			w.Write([]byte(`
			{
   "metadata": {
      "guid": "4ae5d97e-d1c8-49a2-8c73-466bc307f81f",
      "url": "/v2/spaces/4ae5d97e-d1c8-49a2-8c73-466bc307f81f",
   },
   "entity": {
      "name": "pcfdev-space",
      "organization_url": "/v2/organizations/eddb9917-be02-43f2-a625-cf88abaca606"
   }
}
			`))
		}
	})

	http.HandleFunc("/v2/organizations/eddb9917-be02-43f2-a625-cf88abaca606", func(w http.ResponseWriter, r *http.Request) {
		if (r.Method == "GET") {
			w.Write([]byte(`
			{
   "metadata": {
      "guid": "eddb9917-be02-43f2-a625-cf88abaca606",
      "url": "/v2/organizations/eddb9917-be02-43f2-a625-cf88abaca606",
   },
   "entity": {
      "name": "pcfdev-org"
   }
}
			`))
		}
	})

	http.HandleFunc("/v2/service_brokers/0761af0d-4d5a-4aab-a288-39b40a3a09d7", func(w http.ResponseWriter, r *http.Request) {
		if (r.Method == "GET") {
			w.Write([]byte(`
{
   "metadata": {
      "guid": "0761af0d-4d5a-4aab-a288-39b40a3a09d7",
      "url": "/v2/service_brokers/0761af0d-4d5a-4aab-a288-39b40a3a09d7",
      "created_at": "2016-10-31T15:31:38Z",
      "updated_at": null
   },
   "entity": {
      "name": "p-mysql",
      "broker_url": "http://mysql-broker.local.pcfdev.io",
      "auth_username": "admin",
      "space_guid": null
   }
}			`))
		}
	})

	http.HandleFunc("/v2/service_instances/7932fe33-e3ac-4cff-9e08-980313230a89/service_bindings/" + BINDING_GUID, func(w http.ResponseWriter, r *http.Request) {
		var body []byte
		r.Body.Read(body)
		var reqBody struct {
			plan_id    string
			service_id string
		}

		err := json.Unmarshal(body, &reqBody)

		if err == nil {

			if (r.Method == "PUT" && reqBody.plan_id == "40d4bcfe-fccd-4b0a-9e41-e518f9fe4001" && reqBody.service_id == "6544d67e-ea8c-42b4-a528-5953d0a0552f") {
				w.Write([]byte(`
{
  "credentials": {
    "http_api_uris": [
      "https://apcfdev:utgs9v182ahdk4aptakhnu0csl@rabbitmq-management.local.pcfdev.io/api/"
    ],
    "ssl": false,
    "dashboard_url": "https://rabbitmq-management.local.pcfdev.io/#/login/apcfdev/utgs9v182ahdk4aptakhnu0csl",
    "password": "utgs9v182ahdk4aptakhnu0csl",
    "protocols": {
      "management": {
        "path": "/api/",
        "ssl": false,
        "hosts": [
          "rabbitmq.local.pcfdev.io"
        ],
        "password": "utgs9v182ahdk4aptakhnu0csl",
        "username": "apcfdev",
        "port": 15672,
        "host": "rabbitmq.local.pcfdev.io",
        "uri": "http://apcfdev:utgs9v182ahdk4aptakhnu0csl@rabbitmq.local.pcfdev.io:15672/api/",
        "uris": [
          "http://apcfdev:utgs9v182ahdk4aptakhnu0csl@rabbitmq.local.pcfdev.io:15672/api/"
        ]
      },
      "amqp": {
        "vhost": "4552769e-f3a9-4856-8f14-3fd98759cb0b",
        "username": "apcfdev",
        "password": "utgs9v182ahdk4aptakhnu0csl",
        "port": 5672,
        "host": "rabbitmq.local.pcfdev.io",
        "hosts": [
          "rabbitmq.local.pcfdev.io"
        ],
        "ssl": false,
        "uri": "amqp://apcfdev:utgs9v182ahdk4aptakhnu0csl@rabbitmq.local.pcfdev.io:5672/4552769e-f3a9-4856-8f14-3fd98759cb0b",
        "uris": [
          "amqp://apcfdev:utgs9v182ahdk4aptakhnu0csl@rabbitmq.local.pcfdev.io:5672/4552769e-f3a9-4856-8f14-3fd98759cb0b"
        ]
      }
    },
    "username": "apcfdev",
    "hostname": "rabbitmq.local.pcfdev.io",
    "hostnames": [
      "rabbitmq.local.pcfdev.io"
    ],
    "vhost": "4552769e-f3a9-4856-8f14-3fd98759cb0b",
    "http_api_uri": "https://apcfdev:utgs9v182ahdk4aptakhnu0csl@rabbitmq-management.local.pcfdev.io/api/",
    "uri": "amqp://apcfdev:utgs9v182ahdk4aptakhnu0csl@rabbitmq.local.pcfdev.io/4552769e-f3a9-4856-8f14-3fd98759cb0b",
    "uris": [
      "amqp://apcfdev:utgs9v182ahdk4aptakhnu0csl@rabbitmq.local.pcfdev.io/4552769e-f3a9-4856-8f14-3fd98759cb0b"
    ]
  }
}`))
			}
		}

	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("PCF Dev Test VM"))
	})

	if err := http.ListenAndServe(":80", nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
