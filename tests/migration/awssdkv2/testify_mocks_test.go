/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// A mockery mock of a v1 iface interface is rendered afresh against the
// regenerated one, and the calls naming its methods gain the context.
func TestMigrateMockeryMockOfIface(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.GoProject("meter",
			test.GoMod(`
				module example.com/meter

				go 1.24

				require (
					github.com/aws/aws-sdk-go v1.55.8
					github.com/stretchr/testify v1.10.0
				)
			`, `
				module example.com/meter

				go 1.24

				require (
					github.com/stretchr/testify v1.10.0
					github.com/aws/aws-sdk-go-v2 v1.47.3
					github.com/aws/aws-sdk-go-v2/service/marketplacemetering v1.46.2
				)

				require (
					github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.6 // indirect
					github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.6 // indirect
					github.com/aws/smithy-go v1.28.5 // indirect
				)
			`),
			test.Golang(`
				package meter

				import (
					"github.com/aws/aws-sdk-go/aws"
					"github.com/aws/aws-sdk-go/service/marketplacemetering"
					"github.com/aws/aws-sdk-go/service/marketplacemetering/marketplacemeteringiface"
				)

				type Meter struct {
					Client marketplacemeteringiface.MarketplaceMeteringAPI
				}

				func (m *Meter) Customer(token string) (string, error) {
					out, err := m.Client.ResolveCustomer(&marketplacemetering.ResolveCustomerInput{RegistrationToken: aws.String(token)})
					if err != nil {
						return "", err
					}
					return aws.StringValue(out.CustomerIdentifier), nil
				}
			`, `
				package meter

				import (
					"context"

					"github.com/aws/aws-sdk-go-v2/aws"
					"github.com/aws/aws-sdk-go-v2/service/marketplacemetering"
					"example.com/meter/internal/awsiface/marketplacemeteringiface"
				)

				type Meter struct {
					Client marketplacemeteringiface.MarketplaceMeteringAPI
				}

				func (m *Meter) Customer(token string) (string, error) {
					out, err := m.Client.ResolveCustomer(context.TODO(), &marketplacemetering.ResolveCustomerInput{RegistrationToken: aws.String(token)})
					if err != nil {
						return "", err
					}
					return aws.ToString(out.CustomerIdentifier), nil
				}
			`).WithPath("meter.go"),
			test.Golang(mockeryBefore, mockeryAfter).WithPath("mocks/MarketplaceMeteringAPI.go"),
			test.Golang(`
				package meter

				import (
					"context"
					"testing"

					"example.com/meter/mocks"
					"github.com/aws/aws-sdk-go/aws"
					"github.com/aws/aws-sdk-go/service/marketplacemetering"
				)

				func TestCustomer(t *testing.T) {
					client := mocks.NewMarketplaceMeteringAPI(t)
					client.On("ResolveCustomer", &marketplacemetering.ResolveCustomerInput{RegistrationToken: aws.String("t")}).Return(&marketplacemetering.ResolveCustomerOutput{CustomerIdentifier: aws.String("c")}, nil).Once()
					client.On("MeterUsageWithContext", context.Background(), nil).Return(nil, nil)
					m := &Meter{Client: client}
					if c, _ := m.Customer("t"); c != "c" {
						t.Fatal(c)
					}
					client.AssertNumberOfCalls(t, "ResolveCustomer", 1)
					client.AssertNotCalled(t, "RegisterUsage", nil)
				}
			`, `
				package meter

				import (
					"context"
					"testing"

					"github.com/aws/aws-sdk-go-v2/aws"
					"github.com/aws/aws-sdk-go-v2/service/marketplacemetering"
					"github.com/stretchr/testify/mock"

					"example.com/meter/mocks"
				)

				func TestCustomer(t *testing.T) {
					client := mocks.NewMarketplaceMeteringAPI(t)
					client.On("ResolveCustomer", mock.Anything, &marketplacemetering.ResolveCustomerInput{RegistrationToken: aws.String("t")}).Return(&marketplacemetering.ResolveCustomerOutput{CustomerIdentifier: aws.String("c")}, nil).Once()
					client.On("MeterUsage", context.Background(), nil).Return(nil, nil)
					m := &Meter{Client: client}
					if c, _ := m.Customer("t"); c != "c" {
						t.Fatal(c)
					}
					client.AssertNumberOfCalls(t, "ResolveCustomer", 1)
					client.AssertNotCalled(t, "RegisterUsage", mock.Anything, nil)
				}
			`).WithPath("meter_test.go"),
			test.Generated("internal/awsiface/marketplacemeteringiface/zz_awssdkv2_iface.go", meteringIface),
		),
	)
}

// A Run stub reads the call's arguments by position, which the context v2
// passes first shifts, so the test is commented out for porting by hand.
func TestMigrateMockeryRunStubCommentsTestOut(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&awssdkv2.MigrateAwsSdkGoToV2{}).RewriteRun(t,
		test.GoProject("meter",
			test.GoMod(`
				module example.com/meter

				go 1.24

				require (
					github.com/aws/aws-sdk-go v1.55.8
					github.com/stretchr/testify v1.10.0
				)
			`, `
				module example.com/meter

				go 1.24

				require (
					github.com/stretchr/testify v1.10.0
					github.com/aws/aws-sdk-go-v2 v1.47.3
					github.com/aws/aws-sdk-go-v2/service/marketplacemetering v1.46.2
				)

				require (
					github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.6 // indirect
					github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.6 // indirect
					github.com/aws/smithy-go v1.28.5 // indirect
				)
			`),
			test.Golang(`
				package meter

				import (
					"github.com/aws/aws-sdk-go/aws"
					"github.com/aws/aws-sdk-go/service/marketplacemetering"
					"github.com/aws/aws-sdk-go/service/marketplacemetering/marketplacemeteringiface"
				)

				type Meter struct {
					Client marketplacemeteringiface.MarketplaceMeteringAPI
				}

				func (m *Meter) Customer(token string) (string, error) {
					out, err := m.Client.ResolveCustomer(&marketplacemetering.ResolveCustomerInput{RegistrationToken: aws.String(token)})
					if err != nil {
						return "", err
					}
					return aws.StringValue(out.CustomerIdentifier), nil
				}
			`, `
				package meter

				import (
					"context"

					"github.com/aws/aws-sdk-go-v2/aws"
					"github.com/aws/aws-sdk-go-v2/service/marketplacemetering"
					"example.com/meter/internal/awsiface/marketplacemeteringiface"
				)

				type Meter struct {
					Client marketplacemeteringiface.MarketplaceMeteringAPI
				}

				func (m *Meter) Customer(token string) (string, error) {
					out, err := m.Client.ResolveCustomer(context.TODO(), &marketplacemetering.ResolveCustomerInput{RegistrationToken: aws.String(token)})
					if err != nil {
						return "", err
					}
					return aws.ToString(out.CustomerIdentifier), nil
				}
			`).WithPath("meter.go"),
			test.Golang(mockeryBefore, mockeryAfter).WithPath("mocks/MarketplaceMeteringAPI.go"),
			test.Golang(`
				package meter

				import (
					"testing"

					"example.com/meter/mocks"
					"github.com/aws/aws-sdk-go/aws"
					"github.com/aws/aws-sdk-go/service/marketplacemetering"
					"github.com/stretchr/testify/mock"
				)

				func TestCustomer(t *testing.T) {
					client := mocks.NewMarketplaceMeteringAPI(t)
					client.On("ResolveCustomer", mock.Anything).Run(func(args mock.Arguments) {
						_ = args.Get(0).(*marketplacemetering.ResolveCustomerInput)
					}).Return(&marketplacemetering.ResolveCustomerOutput{CustomerIdentifier: aws.String("c")}, nil)
					m := &Meter{Client: client}
					if c, _ := m.Customer("t"); c != "c" {
						t.Fatal(c)
					}
				}
			`, `
				package meter

				// MigrateAwsSdkGoToV2 commented this test out: a Run stub on a mockery expectation, reading arguments that v2 shifts by the context it passes first.
				// It tests aws-sdk-go v1 in a way v2 has no counterpart for; port it to the
				// migrated code and restore it.
				//
				// import (
				// 	"testing"
				//
				// 	"example.com/meter/mocks"
				// 	"github.com/aws/aws-sdk-go/aws"
				// 	"github.com/aws/aws-sdk-go/service/marketplacemetering"
				// 	"github.com/stretchr/testify/mock"
				// )
				//
				// func TestCustomer(t *testing.T) {
				// 	client := mocks.NewMarketplaceMeteringAPI(t)
				// 	client.On("ResolveCustomer", mock.Anything).Run(func(args mock.Arguments) {
				// 		_ = args.Get(0).(*marketplacemetering.ResolveCustomerInput)
				// 	}).Return(&marketplacemetering.ResolveCustomerOutput{CustomerIdentifier: aws.String("c")}, nil)
				// 	m := &Meter{Client: client}
				// 	if c, _ := m.Customer("t"); c != "c" {
				// 		t.Fatal(c)
				// 	}
				// }
			`).WithPath("meter_test.go"),
			test.Generated("internal/awsiface/marketplacemeteringiface/zz_awssdkv2_iface.go", meteringIface),
		),
	)
}

const mockeryBefore = `
				// Code generated by mockery v2.20.0. DO NOT EDIT.

				package mocks

				import (
					context "context"

					request "github.com/aws/aws-sdk-go/aws/request"
					mock "github.com/stretchr/testify/mock"

					marketplacemetering "github.com/aws/aws-sdk-go/service/marketplacemetering"
				)

				// MarketplaceMeteringAPI is an autogenerated mock type for the MarketplaceMeteringAPI type
				type MarketplaceMeteringAPI struct {
					mock.Mock
				}

				// ResolveCustomerRequest provides a mock function with given fields: _a0
				func (_m *MarketplaceMeteringAPI) ResolveCustomerRequest(_a0 *marketplacemetering.ResolveCustomerInput) (*request.Request, *marketplacemetering.ResolveCustomerOutput) {
					ret := _m.Called(_a0)

					var r0 *request.Request
					if rf, ok := ret.Get(0).(func(*marketplacemetering.ResolveCustomerInput) *request.Request); ok {
						r0 = rf(_a0)
					} else {
						if ret.Get(0) != nil {
							r0 = ret.Get(0).(*request.Request)
						}
					}

					var r1 *marketplacemetering.ResolveCustomerOutput
					if rf, ok := ret.Get(1).(func(*marketplacemetering.ResolveCustomerInput) *marketplacemetering.ResolveCustomerOutput); ok {
						r1 = rf(_a0)
					} else {
						if ret.Get(1) != nil {
							r1 = ret.Get(1).(*marketplacemetering.ResolveCustomerOutput)
						}
					}

					return r0, r1
				}

				// ResolveCustomerWithContext provides a mock function with given fields: _a0, _a1, _a2
				func (_m *MarketplaceMeteringAPI) ResolveCustomerWithContext(_a0 context.Context, _a1 *marketplacemetering.ResolveCustomerInput, _a2 ...request.Option) (*marketplacemetering.ResolveCustomerOutput, error) {
					ret := _m.Called(_a0, _a1)
					var r0 *marketplacemetering.ResolveCustomerOutput
					if ret.Get(0) != nil {
						r0 = ret.Get(0).(*marketplacemetering.ResolveCustomerOutput)
					}
					return r0, ret.Error(1)
				}

				type mockConstructorTestingTNewMarketplaceMeteringAPI interface {
					mock.TestingT
					Cleanup(func())
				}

				// NewMarketplaceMeteringAPI creates a new instance of MarketplaceMeteringAPI.
				func NewMarketplaceMeteringAPI(t mockConstructorTestingTNewMarketplaceMeteringAPI) *MarketplaceMeteringAPI {
					mock := &MarketplaceMeteringAPI{}
					mock.Mock.Test(t)

					t.Cleanup(func() { mock.AssertExpectations(t) })

					return mock
				}
			`

const mockeryAfter = `
				// Code generated by mockery v2.20.0. DO NOT EDIT.

				package mocks

				import (
					context "context"

					mock "github.com/stretchr/testify/mock"

					marketplacemetering "github.com/aws/aws-sdk-go-v2/service/marketplacemetering"
				)

				// MarketplaceMeteringAPI is an autogenerated mock type for the MarketplaceMeteringAPI type
				type MarketplaceMeteringAPI struct {
					mock.Mock
				}

				// BatchMeterUsage provides a mock function with given fields: ctx, params, optFns
				func (_m *MarketplaceMeteringAPI) BatchMeterUsage(ctx context.Context, params *marketplacemetering.BatchMeterUsageInput, optFns ...func(*marketplacemetering.Options)) (*marketplacemetering.BatchMeterUsageOutput, error) {
					_va := make([]interface{}, len(optFns))
					for _i := range optFns {
						_va[_i] = optFns[_i]
					}
					var _ca []interface{}
					_ca = append(_ca, ctx, params)
					_ca = append(_ca, _va...)
					ret := _m.Called(_ca...)

					var r0 *marketplacemetering.BatchMeterUsageOutput
					if rf, ok := ret.Get(0).(func(context.Context, *marketplacemetering.BatchMeterUsageInput, ...func(*marketplacemetering.Options)) *marketplacemetering.BatchMeterUsageOutput); ok {
						r0 = rf(ctx, params, optFns...)
					} else {
						if ret.Get(0) != nil {
							r0 = ret.Get(0).(*marketplacemetering.BatchMeterUsageOutput)
						}
					}

					var r1 error
					if rf, ok := ret.Get(1).(func(context.Context, *marketplacemetering.BatchMeterUsageInput, ...func(*marketplacemetering.Options)) error); ok {
						r1 = rf(ctx, params, optFns...)
					} else {
						r1 = ret.Error(1)
					}

					return r0, r1
				}

				// MeterUsage provides a mock function with given fields: ctx, params, optFns
				func (_m *MarketplaceMeteringAPI) MeterUsage(ctx context.Context, params *marketplacemetering.MeterUsageInput, optFns ...func(*marketplacemetering.Options)) (*marketplacemetering.MeterUsageOutput, error) {
					_va := make([]interface{}, len(optFns))
					for _i := range optFns {
						_va[_i] = optFns[_i]
					}
					var _ca []interface{}
					_ca = append(_ca, ctx, params)
					_ca = append(_ca, _va...)
					ret := _m.Called(_ca...)

					var r0 *marketplacemetering.MeterUsageOutput
					if rf, ok := ret.Get(0).(func(context.Context, *marketplacemetering.MeterUsageInput, ...func(*marketplacemetering.Options)) *marketplacemetering.MeterUsageOutput); ok {
						r0 = rf(ctx, params, optFns...)
					} else {
						if ret.Get(0) != nil {
							r0 = ret.Get(0).(*marketplacemetering.MeterUsageOutput)
						}
					}

					var r1 error
					if rf, ok := ret.Get(1).(func(context.Context, *marketplacemetering.MeterUsageInput, ...func(*marketplacemetering.Options)) error); ok {
						r1 = rf(ctx, params, optFns...)
					} else {
						r1 = ret.Error(1)
					}

					return r0, r1
				}

				// RegisterUsage provides a mock function with given fields: ctx, params, optFns
				func (_m *MarketplaceMeteringAPI) RegisterUsage(ctx context.Context, params *marketplacemetering.RegisterUsageInput, optFns ...func(*marketplacemetering.Options)) (*marketplacemetering.RegisterUsageOutput, error) {
					_va := make([]interface{}, len(optFns))
					for _i := range optFns {
						_va[_i] = optFns[_i]
					}
					var _ca []interface{}
					_ca = append(_ca, ctx, params)
					_ca = append(_ca, _va...)
					ret := _m.Called(_ca...)

					var r0 *marketplacemetering.RegisterUsageOutput
					if rf, ok := ret.Get(0).(func(context.Context, *marketplacemetering.RegisterUsageInput, ...func(*marketplacemetering.Options)) *marketplacemetering.RegisterUsageOutput); ok {
						r0 = rf(ctx, params, optFns...)
					} else {
						if ret.Get(0) != nil {
							r0 = ret.Get(0).(*marketplacemetering.RegisterUsageOutput)
						}
					}

					var r1 error
					if rf, ok := ret.Get(1).(func(context.Context, *marketplacemetering.RegisterUsageInput, ...func(*marketplacemetering.Options)) error); ok {
						r1 = rf(ctx, params, optFns...)
					} else {
						r1 = ret.Error(1)
					}

					return r0, r1
				}

				// ResolveCustomer provides a mock function with given fields: ctx, params, optFns
				func (_m *MarketplaceMeteringAPI) ResolveCustomer(ctx context.Context, params *marketplacemetering.ResolveCustomerInput, optFns ...func(*marketplacemetering.Options)) (*marketplacemetering.ResolveCustomerOutput, error) {
					_va := make([]interface{}, len(optFns))
					for _i := range optFns {
						_va[_i] = optFns[_i]
					}
					var _ca []interface{}
					_ca = append(_ca, ctx, params)
					_ca = append(_ca, _va...)
					ret := _m.Called(_ca...)

					var r0 *marketplacemetering.ResolveCustomerOutput
					if rf, ok := ret.Get(0).(func(context.Context, *marketplacemetering.ResolveCustomerInput, ...func(*marketplacemetering.Options)) *marketplacemetering.ResolveCustomerOutput); ok {
						r0 = rf(ctx, params, optFns...)
					} else {
						if ret.Get(0) != nil {
							r0 = ret.Get(0).(*marketplacemetering.ResolveCustomerOutput)
						}
					}

					var r1 error
					if rf, ok := ret.Get(1).(func(context.Context, *marketplacemetering.ResolveCustomerInput, ...func(*marketplacemetering.Options)) error); ok {
						r1 = rf(ctx, params, optFns...)
					} else {
						r1 = ret.Error(1)
					}

					return r0, r1
				}

				type mockConstructorTestingTNewMarketplaceMeteringAPI interface {
					mock.TestingT
					Cleanup(func())
				}

				// NewMarketplaceMeteringAPI creates a new instance of MarketplaceMeteringAPI.
				func NewMarketplaceMeteringAPI(t mockConstructorTestingTNewMarketplaceMeteringAPI) *MarketplaceMeteringAPI {
					mock := &MarketplaceMeteringAPI{}
					mock.Mock.Test(t)

					t.Cleanup(func() { mock.AssertExpectations(t) })

					return mock
				}
			`
