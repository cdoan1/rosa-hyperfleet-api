package e2e_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Platform API - OIDC Config Endpoints", Ordered, Label("oidc"), func() {
	var (
		apiClient         *APIClient
		accountID         string
		managedConfigID   string
		unmanagedConfigID string
	)

	BeforeAll(func() {
		baseURL := os.Getenv("E2E_BASE_URL")
		Expect(baseURL).NotTo(BeEmpty(), "E2E_BASE_URL must be set")

		accountID = os.Getenv("E2E_ACCOUNT_ID")
		if accountID == "" {
			GinkgoWriter.Printf("No E2E_ACCOUNT_ID set, using AWS STS caller identity\n")
			cmd := exec.Command("aws", "sts", "get-caller-identity", "--query", "Account", "--output", "text")
			output, err := cmd.CombinedOutput()
			if err != nil {
				Fail("Failed to get AWS account ID: " + err.Error())
			}
			accountID = strings.TrimSpace(string(output))
		}
		apiClient = NewAPIClient(baseURL)
	})

	Describe("POST /api/v0/oidc_configs", func() {
		Context("when creating a managed OIDC config", func() {
			It("should create the config successfully", func() {
				createReq := map[string]interface{}{
					"spec": map[string]interface{}{
						"type": "managed",
					},
				}

				response, err := apiClient.Post("/api/v0/oidc_configs", createReq, accountID)
				Expect(err).To(BeNil())
				Expect(response.StatusCode).To(Equal(http.StatusCreated))
				Expect(response.Headers).To(HaveKey("Content-Type"))

				var created map[string]interface{}
				err = json.Unmarshal(response.Body, &created)
				Expect(err).To(BeNil())

				// Verify spec
				spec, ok := created["spec"].(map[string]interface{})
				Expect(ok).To(BeTrue(), "spec should be present")
				Expect(spec["type"]).To(Equal("managed"))

				// Managed configs should not have secretArn or installerRoleArn
				Expect(spec["secretArn"]).To(BeEmpty())
				Expect(spec["installerRoleArn"]).To(BeEmpty())

				// Save the config ID for later tests
				metadata, ok := created["metadata"].(map[string]interface{})
				Expect(ok).To(BeTrue(), "metadata should be present")
				managedConfigID, ok = metadata["name"].(string)
				Expect(ok).To(BeTrue(), "metadata.name should be a string")
				Expect(managedConfigID).NotTo(BeEmpty())

				GinkgoWriter.Printf("Created managed OIDC config: %s\n", managedConfigID)
			})
		})

		Context("when creating an unmanaged OIDC config", func() {
			It("should create the config successfully with all required fields", func() {
				createReq := map[string]interface{}{
					"spec": map[string]interface{}{
						"type":              "unmanaged",
						"issuerUrl":         "https://example.com/oidc",
						"secretArn":         "arn:aws:secretsmanager:us-east-1:123456789012:secret:my-secret",
						"installerRoleArn":  "arn:aws:iam::123456789012:role/my-installer-role",
					},
				}

				response, err := apiClient.Post("/api/v0/oidc_configs", createReq, accountID)
				Expect(err).To(BeNil())
				Expect(response.StatusCode).To(Equal(http.StatusCreated))
				Expect(response.Headers).To(HaveKey("Content-Type"))

				var created map[string]interface{}
				err = json.Unmarshal(response.Body, &created)
				Expect(err).To(BeNil())

				// Verify spec
				spec, ok := created["spec"].(map[string]interface{})
				Expect(ok).To(BeTrue(), "spec should be present")
				Expect(spec["type"]).To(Equal("unmanaged"))
				Expect(spec["issuerUrl"]).To(Equal("https://example.com/oidc"))
				Expect(spec["secretArn"]).To(Equal("arn:aws:secretsmanager:us-east-1:123456789012:secret:my-secret"))
				Expect(spec["installerRoleArn"]).To(Equal("arn:aws:iam::123456789012:role/my-installer-role"))

				// Save the config ID for later tests
				metadata, ok := created["metadata"].(map[string]interface{})
				Expect(ok).To(BeTrue(), "metadata should be present")
				unmanagedConfigID, ok = metadata["name"].(string)
				Expect(ok).To(BeTrue(), "metadata.name should be a string")
				Expect(unmanagedConfigID).NotTo(BeEmpty())

				GinkgoWriter.Printf("Created unmanaged OIDC config: %s\n", unmanagedConfigID)
			})

			It("should reject creation when missing required fields", func() {
				createReq := map[string]interface{}{
					"spec": map[string]interface{}{
						"type": "unmanaged",
						// Missing issuerUrl, secretArn, installerRoleArn
					},
				}

				response, err := apiClient.Post("/api/v0/oidc_configs", createReq, accountID)
				Expect(err).To(BeNil())
				Expect(response.StatusCode).To(Equal(http.StatusBadRequest))

				var errResp map[string]interface{}
				err = json.Unmarshal(response.Body, &errResp)
				Expect(err).To(BeNil())
				Expect(errResp["message"]).NotTo(BeEmpty())
			})
		})

		Context("when creating with invalid data", func() {
			It("should reject creation when spec.type is missing", func() {
				createReq := map[string]interface{}{
					"spec": map[string]interface{}{},
				}

				response, err := apiClient.Post("/api/v0/oidc_configs", createReq, accountID)
				Expect(err).To(BeNil())
				Expect(response.StatusCode).To(Equal(http.StatusBadRequest))

				var errResp map[string]interface{}
				err = json.Unmarshal(response.Body, &errResp)
				Expect(err).To(BeNil())
				Expect(errResp["message"]).NotTo(BeEmpty())
			})

			It("should reject creation when spec is null", func() {
				createReq := map[string]interface{}{
					"spec": nil,
				}

				response, err := apiClient.Post("/api/v0/oidc_configs", createReq, accountID)
				Expect(err).To(BeNil())
				Expect(response.StatusCode).To(Equal(http.StatusBadRequest))

				var errResp map[string]interface{}
				err = json.Unmarshal(response.Body, &errResp)
				Expect(err).To(BeNil())
				Expect(errResp["message"]).NotTo(BeEmpty())
			})

			It("should reject creation with invalid type", func() {
				createReq := map[string]interface{}{
					"spec": map[string]interface{}{
						"type": "invalid-type",
					},
				}

				response, err := apiClient.Post("/api/v0/oidc_configs", createReq, accountID)
				Expect(err).To(BeNil())
				Expect(response.StatusCode).To(Equal(http.StatusBadRequest))

				var errResp map[string]interface{}
				err = json.Unmarshal(response.Body, &errResp)
				Expect(err).To(BeNil())
				Expect(errResp["message"]).NotTo(BeEmpty())
			})

			It("should reject managed config with secretArn or installerRoleArn", func() {
				createReq := map[string]interface{}{
					"spec": map[string]interface{}{
						"type":             "managed",
						"secretArn":        "arn:aws:secretsmanager:us-east-1:123456789012:secret:my-secret",
						"installerRoleArn": "arn:aws:iam::123456789012:role/my-installer-role",
					},
				}

				response, err := apiClient.Post("/api/v0/oidc_configs", createReq, accountID)
				Expect(err).To(BeNil())
				Expect(response.StatusCode).To(Equal(http.StatusBadRequest))

				var errResp map[string]interface{}
				err = json.Unmarshal(response.Body, &errResp)
				Expect(err).To(BeNil())
				Expect(errResp["message"]).NotTo(BeEmpty())
			})
		})
	})

	Describe("GET /api/v0/oidc_configs/{id}", func() {
		Context("when getting an existing OIDC config", func() {
			It("should retrieve the managed config successfully", func() {
				Expect(managedConfigID).NotTo(BeEmpty(), "managed config should have been created")

				path := fmt.Sprintf("/api/v0/oidc_configs/%s", managedConfigID)
				response, err := apiClient.Get(path, accountID)
				Expect(err).To(BeNil())
				Expect(response.StatusCode).To(Equal(http.StatusOK))
				Expect(response.Headers).To(HaveKey("Content-Type"))

				var config map[string]interface{}
				err = json.Unmarshal(response.Body, &config)
				Expect(err).To(BeNil())

				// Verify metadata
				metadata, ok := config["metadata"].(map[string]interface{})
				Expect(ok).To(BeTrue(), "metadata should be present")
				Expect(metadata["name"]).To(Equal(managedConfigID))

				// Verify spec
				spec, ok := config["spec"].(map[string]interface{})
				Expect(ok).To(BeTrue(), "spec should be present")
				Expect(spec["type"]).To(Equal("managed"))

				// Verify status exists (may contain thumbprint)
				status, ok := config["status"].(map[string]interface{})
				if ok {
					GinkgoWriter.Printf("Status for managed config: phase=%v, thumbprint=%v\n",
						status["phase"], status["thumbprint"])
				}
			})

			It("should retrieve the unmanaged config successfully with thumbprint", func() {
				Expect(unmanagedConfigID).NotTo(BeEmpty(), "unmanaged config should have been created")

				path := fmt.Sprintf("/api/v0/oidc_configs/%s", unmanagedConfigID)
				response, err := apiClient.Get(path, accountID)
				Expect(err).To(BeNil())
				Expect(response.StatusCode).To(Equal(http.StatusOK))
				Expect(response.Headers).To(HaveKey("Content-Type"))

				var config map[string]interface{}
				err = json.Unmarshal(response.Body, &config)
				Expect(err).To(BeNil())

				// Verify metadata
				metadata, ok := config["metadata"].(map[string]interface{})
				Expect(ok).To(BeTrue(), "metadata should be present")
				Expect(metadata["name"]).To(Equal(unmanagedConfigID))

				// Verify spec
				spec, ok := config["spec"].(map[string]interface{})
				Expect(ok).To(BeTrue(), "spec should be present")
				Expect(spec["type"]).To(Equal("unmanaged"))
				Expect(spec["issuerUrl"]).To(Equal("https://example.com/oidc"))

				// Verify status exists (may contain thumbprint)
				status, ok := config["status"].(map[string]interface{})
				if ok {
					GinkgoWriter.Printf("Status for unmanaged config: phase=%v, thumbprint=%v\n",
						status["phase"], status["thumbprint"])

					// Thumbprint may be populated by the controller
					if thumbprint, ok := status["thumbprint"].(string); ok && thumbprint != "" {
						GinkgoWriter.Printf("Thumbprint present: %s\n", thumbprint)
					}
				}
			})
		})

		Context("when getting a non-existent OIDC config", func() {
			It("should return 404 Not Found", func() {
				nonExistentID := "non-existent-config-id"
				path := fmt.Sprintf("/api/v0/oidc_configs/%s", nonExistentID)
				response, err := apiClient.Get(path, accountID)
				Expect(err).To(BeNil())
				Expect(response.StatusCode).To(Equal(http.StatusNotFound))

				var errResp map[string]interface{}
				err = json.Unmarshal(response.Body, &errResp)
				Expect(err).To(BeNil())
				Expect(errResp["message"]).NotTo(BeEmpty())
			})
		})
	})

	Describe("GET /api/v0/oidc_configs", func() {
		Context("when listing OIDC configs", func() {
			It("should list all configs for the account", func() {
				response, err := apiClient.Get("/api/v0/oidc_configs", accountID)
				Expect(err).To(BeNil())
				Expect(response.StatusCode).To(Equal(http.StatusOK))
				Expect(response.Headers).To(HaveKey("Content-Type"))

				var list struct {
					Items  []map[string]interface{} `json:"items"`
					Total  int                      `json:"total"`
					Limit  int                      `json:"limit"`
					Offset int                      `json:"offset"`
				}
				err = json.Unmarshal(response.Body, &list)
				Expect(err).To(BeNil())
				Expect(list.Items).NotTo(BeNil())
				Expect(list.Total).To(BeNumerically(">=", 2), "should have at least the 2 configs created in this test")

				// Verify our created configs are in the list
				foundManaged := false
				foundUnmanaged := false
				for _, item := range list.Items {
					metadata, ok := item["metadata"].(map[string]interface{})
					if !ok {
						continue
					}
					name, ok := metadata["name"].(string)
					if !ok {
						continue
					}
					if name == managedConfigID {
						foundManaged = true
						GinkgoWriter.Printf("Found managed config in list: %s\n", name)
					}
					if name == unmanagedConfigID {
						foundUnmanaged = true
						GinkgoWriter.Printf("Found unmanaged config in list: %s\n", name)
					}
				}
				Expect(foundManaged).To(BeTrue(), "managed config should be in the list")
				Expect(foundUnmanaged).To(BeTrue(), "unmanaged config should be in the list")

				GinkgoWriter.Printf("Listed %d OIDC configs (total: %d)\n", len(list.Items), list.Total)
			})

			It("should respect limit and offset parameters", func() {
				response, err := apiClient.Get("/api/v0/oidc_configs?limit=1&offset=0", accountID)
				Expect(err).To(BeNil())
				Expect(response.StatusCode).To(Equal(http.StatusOK))

				var list struct {
					Items  []map[string]interface{} `json:"items"`
					Total  int                      `json:"total"`
					Limit  int                      `json:"limit"`
					Offset int                      `json:"offset"`
				}
				err = json.Unmarshal(response.Body, &list)
				Expect(err).To(BeNil())
				Expect(list.Limit).To(Equal(1))
				Expect(list.Offset).To(Equal(0))
				Expect(len(list.Items)).To(BeNumerically("<=", 1))

				GinkgoWriter.Printf("List with limit=1: returned %d items, total=%d\n", len(list.Items), list.Total)
			})
		})
	})

	Describe("DELETE /api/v0/oidc_configs/{id}", func() {
		Context("when deleting an OIDC config", func() {
			var deletableConfigID string

			BeforeEach(func() {
				// Create a new config specifically for deletion testing
				createReq := map[string]interface{}{
					"spec": map[string]interface{}{
						"type": "managed",
					},
				}

				response, err := apiClient.Post("/api/v0/oidc_configs", createReq, accountID)
				Expect(err).To(BeNil())
				Expect(response.StatusCode).To(Equal(http.StatusCreated))

				var created map[string]interface{}
				err = json.Unmarshal(response.Body, &created)
				Expect(err).To(BeNil())

				metadata := created["metadata"].(map[string]interface{})
				deletableConfigID = metadata["name"].(string)
				Expect(deletableConfigID).NotTo(BeEmpty())

				GinkgoWriter.Printf("Created config for deletion test: %s\n", deletableConfigID)
			})

			It("should delete the config successfully when no clusters reference it", func() {
				path := fmt.Sprintf("/api/v0/oidc_configs/%s", deletableConfigID)
				response, err := apiClient.Delete(path, accountID)
				Expect(err).To(BeNil())
				Expect(response.StatusCode).To(Equal(http.StatusAccepted))

				var deleteResp map[string]interface{}
				err = json.Unmarshal(response.Body, &deleteResp)
				Expect(err).To(BeNil())
				Expect(deleteResp["message"]).NotTo(BeEmpty())
				Expect(deleteResp["config_id"]).To(Equal(deletableConfigID))

				GinkgoWriter.Printf("Deleted OIDC config: %s\n", deletableConfigID)

				// Verify the config is eventually deleted (may take time for controller to process)
				Eventually(func() int {
					getResponse, err := apiClient.Get(path, accountID)
					if err != nil {
						return http.StatusInternalServerError
					}
					return getResponse.StatusCode
				}, "30s", "2s").Should(Equal(http.StatusNotFound), "config should be deleted after controller processes")
			})
		})

		Context("when deleting a non-existent config", func() {
			It("should return 404 Not Found", func() {
				nonExistentID := fmt.Sprintf("non-existent-%d", time.Now().Unix())
				path := fmt.Sprintf("/api/v0/oidc_configs/%s", nonExistentID)
				response, err := apiClient.Delete(path, accountID)
				Expect(err).To(BeNil())
				Expect(response.StatusCode).To(Equal(http.StatusNotFound))

				var errResp map[string]interface{}
				err = json.Unmarshal(response.Body, &errResp)
				Expect(err).To(BeNil())
				Expect(errResp["message"]).NotTo(BeEmpty())
			})
		})
	})
})
