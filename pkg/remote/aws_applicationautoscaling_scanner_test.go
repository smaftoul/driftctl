package remote

import (
	"testing"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/applicationautoscaling"
	"github.com/cloudskiff/driftctl/mocks"
	"github.com/cloudskiff/driftctl/pkg/filter"
	"github.com/cloudskiff/driftctl/pkg/remote/aws"
	"github.com/cloudskiff/driftctl/pkg/remote/aws/repository"
	"github.com/cloudskiff/driftctl/pkg/remote/cache"
	"github.com/cloudskiff/driftctl/pkg/remote/common"
	remoteerror "github.com/cloudskiff/driftctl/pkg/remote/error"
	"github.com/cloudskiff/driftctl/pkg/resource"
	resourceaws "github.com/cloudskiff/driftctl/pkg/resource/aws"
	"github.com/cloudskiff/driftctl/pkg/terraform"
	"github.com/cloudskiff/driftctl/test"
	"github.com/cloudskiff/driftctl/test/goldenfile"
	testresource "github.com/cloudskiff/driftctl/test/resource"
	terraform2 "github.com/cloudskiff/driftctl/test/terraform"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAppAutoScalingTarget(t *testing.T) {
	tests := []struct {
		test    string
		dirName string
		mocks   func(*repository.MockAppAutoScalingRepository, *mocks.AlerterInterface)
		wantErr error
	}{
		{
			test:    "should return one target",
			dirName: "aws_appautoscaling_target_single",
			mocks: func(client *repository.MockAppAutoScalingRepository, alerter *mocks.AlerterInterface) {
				client.On("ServiceNamespaceValues").Return(applicationautoscaling.ServiceNamespace_Values()).Once()

				client.On("DescribeScalableTargets", "dynamodb").Return([]*applicationautoscaling.ScalableTarget{
					{
						ResourceId:        awssdk.String("table/GameScores"),
						RoleARN:           awssdk.String("arn:aws:iam::533948124879:role/aws-service-role/dynamodb.application-autoscaling.amazonaws.com/AWSServiceRoleForApplicationAutoScaling_DynamoDBTable"),
						ScalableDimension: awssdk.String("dynamodb:table:ReadCapacityUnits"),
						ServiceNamespace:  awssdk.String("dynamodb"),
						MaxCapacity:       awssdk.Int64(100),
						MinCapacity:       awssdk.Int64(5),
					},
				}, nil).Once()

				client.On("DescribeScalableTargets", mock.AnythingOfType("string")).Return([]*applicationautoscaling.ScalableTarget{}, nil).Times(len(applicationautoscaling.ServiceNamespace_Values()) - 1)
			},
			wantErr: nil,
		},
		{
			test:    "should return remote error",
			dirName: "aws_appautoscaling_target_single",
			mocks: func(client *repository.MockAppAutoScalingRepository, alerter *mocks.AlerterInterface) {
				client.On("ServiceNamespaceValues").Return(applicationautoscaling.ServiceNamespace_Values()).Once()

				client.On("DescribeScalableTargets", mock.AnythingOfType("string")).Return(nil, errors.New("remote error")).Once()
			},
			wantErr: remoteerror.NewResourceListingError(errors.New("remote error"), resourceaws.AwsAppAutoscalingTargetResourceType),
		},
	}

	schemaRepository := testresource.InitFakeSchemaRepository("aws", "3.19.0")
	resourceaws.InitResourcesMetadata(schemaRepository)
	factory := terraform.NewTerraformResourceFactory(schemaRepository)
	deserializer := resource.NewDeserializer(factory)

	for _, c := range tests {
		t.Run(c.test, func(tt *testing.T) {
			shouldUpdate := c.dirName == *goldenfile.Update

			sess := session.Must(session.NewSessionWithOptions(session.Options{
				SharedConfigState: session.SharedConfigEnable,
			}))

			scanOptions := ScannerOptions{Deep: true}
			providerLibrary := terraform.NewProviderLibrary()
			remoteLibrary := common.NewRemoteLibrary()

			// Initialize mocks
			alerter := &mocks.AlerterInterface{}
			fakeRepo := &repository.MockAppAutoScalingRepository{}
			c.mocks(fakeRepo, alerter)

			var repo repository.AppAutoScalingRepository = fakeRepo
			providerVersion := "3.19.0"
			realProvider, err := terraform2.InitTestAwsProvider(providerLibrary, providerVersion)
			if err != nil {
				t.Fatal(err)
			}
			provider := terraform2.NewFakeTerraformProvider(realProvider)
			provider.WithResponse(c.dirName)

			// Replace mock by real resources if we are in update mode
			if shouldUpdate {
				err := realProvider.Init()
				if err != nil {
					t.Fatal(err)
				}
				provider.ShouldUpdate()
				repo = repository.NewAppAutoScalingRepository(sess, cache.New(0))
			}

			remoteLibrary.AddEnumerator(aws.NewAppAutoscalingTargetEnumerator(repo, factory))
			remoteLibrary.AddDetailsFetcher(resourceaws.AwsAppAutoscalingTargetResourceType, common.NewGenericDetailsFetcher(resourceaws.AwsAppAutoscalingTargetResourceType, provider, deserializer))

			testFilter := &filter.MockFilter{}
			testFilter.On("IsTypeIgnored", mock.Anything).Return(false)

			s := NewScanner(remoteLibrary, alerter, scanOptions, testFilter)
			got, err := s.Resources()
			if err != nil {
				assert.EqualError(tt, c.wantErr, err.Error())
			} else {
				assert.Equal(tt, err, c.wantErr)
			}

			if err != nil {
				return
			}
			test.TestAgainstGoldenFile(got, resourceaws.AwsAppAutoscalingTargetResourceType, c.dirName, provider, deserializer, shouldUpdate, tt)
			alerter.AssertExpectations(tt)
			fakeRepo.AssertExpectations(tt)
		})
	}
}

func TestAppAutoScalingPolicy(t *testing.T) {
	tests := []struct {
		test    string
		dirName string
		mocks   func(*repository.MockAppAutoScalingRepository, *mocks.AlerterInterface)
		wantErr error
	}{
		{
			test:    "should return one policy",
			dirName: "aws_appautoscaling_policy_single",
			mocks: func(client *repository.MockAppAutoScalingRepository, alerter *mocks.AlerterInterface) {
				client.On("ServiceNamespaceValues").Return(applicationautoscaling.ServiceNamespace_Values()).Once()

				client.On("DescribeScalingPolicies", "dynamodb").Return([]*applicationautoscaling.ScalingPolicy{
					{
						PolicyName:        awssdk.String("DynamoDBReadCapacityUtilization:table/GameScores"),
						ResourceId:        awssdk.String("table/GameScores"),
						ScalableDimension: awssdk.String("dynamodb:table:ReadCapacityUnits"),
						ServiceNamespace:  awssdk.String("dynamodb"),
					},
				}, nil).Once()

				client.On("DescribeScalingPolicies", mock.AnythingOfType("string")).Return([]*applicationautoscaling.ScalingPolicy{}, nil).Times(len(applicationautoscaling.ServiceNamespace_Values()) - 1)
			},
			wantErr: nil,
		},
		{
			test:    "should return remote error",
			dirName: "aws_appautoscaling_policy_single",
			mocks: func(client *repository.MockAppAutoScalingRepository, alerter *mocks.AlerterInterface) {
				client.On("ServiceNamespaceValues").Return(applicationautoscaling.ServiceNamespace_Values()).Once()

				client.On("DescribeScalingPolicies", mock.AnythingOfType("string")).Return(nil, errors.New("remote error")).Once()
			},
			wantErr: remoteerror.NewResourceListingError(errors.New("remote error"), resourceaws.AwsAppAutoscalingPolicyResourceType),
		},
	}

	schemaRepository := testresource.InitFakeSchemaRepository("aws", "3.19.0")
	resourceaws.InitResourcesMetadata(schemaRepository)
	factory := terraform.NewTerraformResourceFactory(schemaRepository)
	deserializer := resource.NewDeserializer(factory)

	for _, c := range tests {
		t.Run(c.test, func(tt *testing.T) {
			shouldUpdate := c.dirName == *goldenfile.Update

			sess := session.Must(session.NewSessionWithOptions(session.Options{
				SharedConfigState: session.SharedConfigEnable,
			}))

			scanOptions := ScannerOptions{Deep: true}
			providerLibrary := terraform.NewProviderLibrary()
			remoteLibrary := common.NewRemoteLibrary()

			// Initialize mocks
			alerter := &mocks.AlerterInterface{}
			fakeRepo := &repository.MockAppAutoScalingRepository{}
			c.mocks(fakeRepo, alerter)

			var repo repository.AppAutoScalingRepository = fakeRepo
			providerVersion := "3.19.0"
			realProvider, err := terraform2.InitTestAwsProvider(providerLibrary, providerVersion)
			if err != nil {
				t.Fatal(err)
			}
			provider := terraform2.NewFakeTerraformProvider(realProvider)
			provider.WithResponse(c.dirName)

			// Replace mock by real resources if we are in update mode
			if shouldUpdate {
				err := realProvider.Init()
				if err != nil {
					t.Fatal(err)
				}
				provider.ShouldUpdate()
				repo = repository.NewAppAutoScalingRepository(sess, cache.New(0))
			}

			remoteLibrary.AddEnumerator(aws.NewAppAutoscalingPolicyEnumerator(repo, factory))
			remoteLibrary.AddDetailsFetcher(resourceaws.AwsAppAutoscalingPolicyResourceType, common.NewGenericDetailsFetcher(resourceaws.AwsAppAutoscalingPolicyResourceType, provider, deserializer))

			testFilter := &filter.MockFilter{}
			testFilter.On("IsTypeIgnored", mock.Anything).Return(false)

			s := NewScanner(remoteLibrary, alerter, scanOptions, testFilter)
			got, err := s.Resources()
			if err != nil {
				assert.EqualError(tt, c.wantErr, err.Error())
			} else {
				assert.Equal(tt, err, c.wantErr)
			}

			if err != nil {
				return
			}
			test.TestAgainstGoldenFile(got, resourceaws.AwsAppAutoscalingPolicyResourceType, c.dirName, provider, deserializer, shouldUpdate, tt)
			alerter.AssertExpectations(tt)
			fakeRepo.AssertExpectations(tt)
		})
	}
}

func TestAppAutoScalingScheduledAction(t *testing.T) {
	dummyError := errors.New("this is an error")

	tests := []struct {
		test           string
		mocks          func(*repository.MockAppAutoScalingRepository, *mocks.AlerterInterface)
		assertExpected func(t *testing.T, got []*resource.Resource)
		wantErr        error
	}{
		{
			test: "should return one scheduled action",
			mocks: func(client *repository.MockAppAutoScalingRepository, alerter *mocks.AlerterInterface) {
				matchServiceNamespaceFunc := func(ns string) bool {
					for _, n := range applicationautoscaling.ServiceNamespace_Values() {
						if n == ns {
							return true
						}
					}
					return false
				}

				client.On("ServiceNamespaceValues").Return(applicationautoscaling.ServiceNamespace_Values()).Once()

				client.On("DescribeScheduledActions", mock.MatchedBy(matchServiceNamespaceFunc)).Return([]*applicationautoscaling.ScheduledAction{
					{
						ScheduledActionName: awssdk.String("action"),
						ResourceId:          awssdk.String("table/GameScores"),
						ScalableDimension:   awssdk.String("dynamodb:table:ReadCapacityUnits"),
						ServiceNamespace:    awssdk.String("dynamodb"),
					},
				}, nil).Once()

				client.On("DescribeScheduledActions", mock.MatchedBy(matchServiceNamespaceFunc)).Return([]*applicationautoscaling.ScheduledAction{}, nil).Times(len(applicationautoscaling.ServiceNamespace_Values()) - 1)
			},
			assertExpected: func(t *testing.T, got []*resource.Resource) {
				assert.Len(t, got, 1)
				assert.Equal(t, "action-dynamodb-table/GameScores", got[0].ResourceId())
				assert.Equal(t, resourceaws.AwsAppAutoscalingScheduledActionResourceType, got[0].ResourceType())
			},
			wantErr: nil,
		},
		{
			test: "should return remote error",
			mocks: func(client *repository.MockAppAutoScalingRepository, alerter *mocks.AlerterInterface) {
				client.On("ServiceNamespaceValues").Return(applicationautoscaling.ServiceNamespace_Values()).Once()

				client.On("DescribeScheduledActions", mock.AnythingOfType("string")).Return(nil, dummyError).Once()
			},
			assertExpected: func(t *testing.T, got []*resource.Resource) {
				assert.Len(t, got, 0)
			},
			wantErr: remoteerror.NewResourceListingError(dummyError, resourceaws.AwsAppAutoscalingScheduledActionResourceType),
		},
	}

	providerVersion := "3.19.0"
	schemaRepository := testresource.InitFakeSchemaRepository("aws", providerVersion)
	resourceaws.InitResourcesMetadata(schemaRepository)
	factory := terraform.NewTerraformResourceFactory(schemaRepository)

	for _, c := range tests {
		t.Run(c.test, func(tt *testing.T) {
			scanOptions := ScannerOptions{}
			remoteLibrary := common.NewRemoteLibrary()

			// Initialize mocks
			alerter := &mocks.AlerterInterface{}
			fakeRepo := &repository.MockAppAutoScalingRepository{}
			c.mocks(fakeRepo, alerter)

			var repo repository.AppAutoScalingRepository = fakeRepo

			remoteLibrary.AddEnumerator(aws.NewAppAutoscalingScheduledActionEnumerator(repo, factory))

			testFilter := &filter.MockFilter{}
			testFilter.On("IsTypeIgnored", mock.Anything).Return(false)

			s := NewScanner(remoteLibrary, alerter, scanOptions, testFilter)
			got, err := s.Resources()
			assert.Equal(tt, err, c.wantErr)
			if err != nil {
				return
			}

			c.assertExpected(tt, got)
			alerter.AssertExpectations(tt)
			fakeRepo.AssertExpectations(tt)
			testFilter.AssertExpectations(tt)
		})
	}
}
