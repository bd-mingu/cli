package api_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/cloudfoundry/cli/cf/api/apifakes"
	"github.com/cloudfoundry/cli/cf/configuration/coreconfig"
	"github.com/cloudfoundry/cli/cf/errors"
	"github.com/cloudfoundry/cli/cf/models"
	"github.com/cloudfoundry/cli/testhelpers/cloudcontrollergateway"
	testconfig "github.com/cloudfoundry/cli/testhelpers/configuration"
	testnet "github.com/cloudfoundry/cli/testhelpers/net"

	. "github.com/cloudfoundry/cli/cf/api"
	. "github.com/cloudfoundry/cli/testhelpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ServiceAuthTokensRepo", func() {
	var (
		testServer  *httptest.Server
		testHandler *testnet.TestHandler
		configRepo  coreconfig.ReadWriter
		repo        CloudControllerServiceAuthTokenRepository
	)

	setupTestServer := func(reqs ...testnet.TestRequest) {
		testServer, testHandler = testnet.NewServer(reqs)
		configRepo.SetAPIEndpoint(testServer.URL)
	}

	BeforeEach(func() {
		configRepo = testconfig.NewRepositoryWithDefaults()

		gateway := cloudcontrollergateway.NewTestCloudControllerGateway(configRepo)
		repo = NewCloudControllerServiceAuthTokenRepository(configRepo, gateway)
	})

	AfterEach(func() {
		testServer.Close()
	})

	Describe("Create", func() {
		It("creates a service auth token", func() {
			setupTestServer(apifakes.NewCloudControllerTestRequest(testnet.TestRequest{
				Method:   "POST",
				Path:     "/v2/service_auth_tokens",
				Matcher:  testnet.RequestBodyMatcher(`{"label":"a label","provider":"a provider","token":"a token"}`),
				Response: testnet.TestResponse{Status: http.StatusCreated},
			}))

			err := repo.Create(models.ServiceAuthTokenFields{
				Label:    "a label",
				Provider: "a provider",
				Token:    "a token",
			})

			Expect(testHandler).To(HaveAllRequestsCalled())
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("FindAll", func() {
		var firstServiceAuthTokenRequest = apifakes.NewCloudControllerTestRequest(testnet.TestRequest{
			Method: "GET",
			Path:   "/v2/service_auth_tokens",
			Response: testnet.TestResponse{
				Status: http.StatusOK,
				Body: `
				{
					"next_url": "/v2/service_auth_tokens?page=2",
					"resources": [
						{
							"metadata": {
								"guid": "mongodb-core-guid"
							},
							"entity": {
								"label": "mongodb",
								"provider": "mongodb-core"
							}
						}
					]
				}`,
			},
		})

		var secondServiceAuthTokenRequest = apifakes.NewCloudControllerTestRequest(testnet.TestRequest{
			Method: "GET",
			Path:   "/v2/service_auth_tokens",
			Response: testnet.TestResponse{
				Status: http.StatusOK,
				Body: `
				{
					"resources": [
						{
							"metadata": {
								"guid": "mysql-core-guid"
							},
							"entity": {
								"label": "mysql",
								"provider": "mysql-core"
							}
						},
						{
							"metadata": {
								"guid": "postgres-core-guid"
							},
							"entity": {
								"label": "postgres",
								"provider": "postgres-core"
							}
						}
					]
				}`,
			},
		})

		BeforeEach(func() {
			setupTestServer(firstServiceAuthTokenRequest, secondServiceAuthTokenRequest)
		})

		It("finds all service auth tokens", func() {
			authTokens, err := repo.FindAll()

			Expect(testHandler).To(HaveAllRequestsCalled())
			Expect(err).NotTo(HaveOccurred())

			Expect(len(authTokens)).To(Equal(3))

			Expect(authTokens[0].Label).To(Equal("mongodb"))
			Expect(authTokens[0].Provider).To(Equal("mongodb-core"))
			Expect(authTokens[0].GUID).To(Equal("mongodb-core-guid"))

			Expect(authTokens[1].Label).To(Equal("mysql"))
			Expect(authTokens[1].Provider).To(Equal("mysql-core"))
			Expect(authTokens[1].GUID).To(Equal("mysql-core-guid"))
		})
	})

	Describe("FindByLabelAndProvider", func() {
		Context("when the auth token exists", func() {
			BeforeEach(func() {
				setupTestServer(apifakes.NewCloudControllerTestRequest(testnet.TestRequest{
					Method: "GET",
					Path:   "/v2/service_auth_tokens?q=label%3Aa-label%3Bprovider%3Aa-provider",
					Response: testnet.TestResponse{
						Status: http.StatusOK,
						Body: `{
					"resources": [{
						"metadata": { "guid": "mysql-core-guid" },
						"entity": {
							"label": "mysql",
							"provider": "mysql-core"
						}
					}]}`,
					},
				}))
			})

			It("returns the auth token", func() {
				serviceAuthToken, err := repo.FindByLabelAndProvider("a-label", "a-provider")

				Expect(testHandler).To(HaveAllRequestsCalled())
				Expect(err).NotTo(HaveOccurred())
				Expect(serviceAuthToken).To(Equal(models.ServiceAuthTokenFields{
					GUID:     "mysql-core-guid",
					Label:    "mysql",
					Provider: "mysql-core",
				}))
			})
		})

		Context("when the auth token does not exist", func() {
			BeforeEach(func() {
				setupTestServer(apifakes.NewCloudControllerTestRequest(testnet.TestRequest{
					Method: "GET",
					Path:   "/v2/service_auth_tokens?q=label%3Aa-label%3Bprovider%3Aa-provider",
					Response: testnet.TestResponse{
						Status: http.StatusOK,
						Body:   `{"resources": []}`},
				}))
			})

			It("returns a ModelNotFoundError", func() {
				_, err := repo.FindByLabelAndProvider("a-label", "a-provider")

				Expect(testHandler).To(HaveAllRequestsCalled())
				Expect(err).To(BeAssignableToTypeOf(&errors.ModelNotFoundError{}))
			})
		})
	})

	Describe("Update", func() {
		It("updates the service auth token", func() {
			setupTestServer(apifakes.NewCloudControllerTestRequest(testnet.TestRequest{
				Method:   "PUT",
				Path:     "/v2/service_auth_tokens/mysql-core-guid",
				Matcher:  testnet.RequestBodyMatcher(`{"token":"a value"}`),
				Response: testnet.TestResponse{Status: http.StatusOK},
			}))

			err := repo.Update(models.ServiceAuthTokenFields{
				GUID:  "mysql-core-guid",
				Token: "a value",
			})

			Expect(testHandler).To(HaveAllRequestsCalled())
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Delete", func() {
		It("deletes the service auth token", func() {

			setupTestServer(apifakes.NewCloudControllerTestRequest(testnet.TestRequest{
				Method:   "DELETE",
				Path:     "/v2/service_auth_tokens/mysql-core-guid",
				Response: testnet.TestResponse{Status: http.StatusOK},
			}))

			err := repo.Delete(models.ServiceAuthTokenFields{
				GUID: "mysql-core-guid",
			})

			Expect(testHandler).To(HaveAllRequestsCalled())
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
