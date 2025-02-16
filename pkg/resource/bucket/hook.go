// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package bucket

import (
	"context"
	"strings"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	svcapitypes "github.com/aws-controllers-k8s/s3-controller/apis/v1alpha1"
	svcsdk "github.com/aws/aws-sdk-go/service/s3"
)

var (
	DefaultAccelerationStatus = svcsdk.BucketAccelerateStatusSuspended
	DefaultRequestPayer       = svcsdk.PayerBucketOwner
	DefaultVersioningStatus   = svcsdk.BucketVersioningStatusSuspended
	DefaultACL                = svcsdk.BucketCannedACLPrivate
)

var (
	CannedACLJoinDelimiter = "|"
)

func (rm *resourceManager) createPutFields(
	ctx context.Context,
	r *resource,
) error {
	// Other configuration options (Replication) require versioning to be
	// enabled before they can be configured
	if r.ko.Spec.Versioning != nil {
		if err := rm.syncVersioning(ctx, r); err != nil {
			return err
		}
	}

	if r.ko.Spec.Accelerate != nil {
		if err := rm.syncAccelerate(ctx, r); err != nil {
			return err
		}
	}
	if r.ko.Spec.CORS != nil {
		if err := rm.syncCORS(ctx, r); err != nil {
			return err
		}
	}
	if r.ko.Spec.Encryption != nil {
		if err := rm.syncEncryption(ctx, r); err != nil {
			return err
		}
	}
	if r.ko.Spec.Lifecycle != nil {
		if err := rm.syncLifecycle(ctx, r); err != nil {
			return err
		}
	}
	if r.ko.Spec.Logging != nil {
		if err := rm.syncLogging(ctx, r); err != nil {
			return err
		}
	}
	if r.ko.Spec.Notification != nil {
		if err := rm.syncNotification(ctx, r); err != nil {
			return err
		}
	}
	if r.ko.Spec.OwnershipControls != nil {
		if err := rm.syncOwnershipControls(ctx, r); err != nil {
			return err
		}
	}
	if r.ko.Spec.Policy != nil {
		if err := rm.syncPolicy(ctx, r); err != nil {
			return err
		}
	}
	if r.ko.Spec.Replication != nil {
		if err := rm.syncReplication(ctx, r); err != nil {
			return err
		}
	}
	if r.ko.Spec.RequestPayment != nil {
		if err := rm.syncRequestPayment(ctx, r); err != nil {
			return err
		}
	}
	if r.ko.Spec.Tagging != nil {
		if err := rm.syncTagging(ctx, r); err != nil {
			return err
		}
	}
	if r.ko.Spec.Website != nil {
		if err := rm.syncWebsite(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

// customUpdateBucket patches each of the resource properties in the backend AWS
// service API and returns a new resource with updated fields.
func (rm *resourceManager) customUpdateBucket(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (updated *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.customUpdateBucket")
	defer exit(err)

	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := desired.ko.DeepCopy()

	rm.setStatusDefaults(ko)

	if delta.DifferentAt("Spec.Accelerate") {
		if err := rm.syncAccelerate(ctx, desired); err != nil {
			return nil, err
		}
	}
	if delta.DifferentAt("Spec.ACL") ||
		delta.DifferentAt("Spec.GrantFullControl") ||
		delta.DifferentAt("Spec.GrantRead") ||
		delta.DifferentAt("Spec.GrantReadACP") ||
		delta.DifferentAt("Spec.GrantWrite") ||
		delta.DifferentAt("Spec.GrantWriteACP") {
		if err := rm.syncACL(ctx, desired); err != nil {
			return nil, err
		}
	}
	if delta.DifferentAt("Spec.CORS") {
		if err := rm.syncCORS(ctx, desired); err != nil {
			return nil, err
		}
	}
	if delta.DifferentAt("Spec.Encryption") {
		if err := rm.syncEncryption(ctx, desired); err != nil {
			return nil, err
		}
	}
	if delta.DifferentAt("Spec.Lifecycle") {
		if err := rm.syncLifecycle(ctx, desired); err != nil {
			return nil, err
		}
	}
	if delta.DifferentAt("Spec.Logging") {
		if err := rm.syncLogging(ctx, desired); err != nil {
			return nil, err
		}
	}
	if delta.DifferentAt("Spec.Notification") {
		if err := rm.syncNotification(ctx, desired); err != nil {
			return nil, err
		}
	}
	if delta.DifferentAt("Spec.OwnershipControls") {
		if err := rm.syncOwnershipControls(ctx, desired); err != nil {
			return nil, err
		}
	}
	if delta.DifferentAt("Spec.Policy") {
		if err := rm.syncPolicy(ctx, desired); err != nil {
			return nil, err
		}
	}
	if delta.DifferentAt("Spec.RequestPayment") {
		if err := rm.syncRequestPayment(ctx, desired); err != nil {
			return nil, err
		}
	}
	if delta.DifferentAt("Spec.Tagging") {
		if err := rm.syncTagging(ctx, desired); err != nil {
			return nil, err
		}
	}
	if delta.DifferentAt("Spec.Website") {
		if err := rm.syncWebsite(ctx, desired); err != nil {
			return nil, err
		}
	}

	// Replication requires versioning be enabled. We check that if we are
	// disabling versioning, that we disable replication first. If we are
	// enabling replication, that we enable versioning first.
	if delta.DifferentAt("Spec.Replication") || delta.DifferentAt("Spec.Versioning") {
		if desired.ko.Spec.Replication == nil || desired.ko.Spec.Replication.Rules == nil {
			if err := rm.syncReplication(ctx, desired); err != nil {
				return nil, err
			}
			if err := rm.syncVersioning(ctx, desired); err != nil {
				return nil, err
			}
		} else {
			if err := rm.syncVersioning(ctx, desired); err != nil {
				return nil, err
			}
			if err := rm.syncReplication(ctx, desired); err != nil {
				return nil, err
			}
		}
	}

	return &resource{ko}, nil
}

// addPutFieldsToSpec will describe each of the Put* fields and add their
// returned values to the Bucket spec.
func (rm *resourceManager) addPutFieldsToSpec(
	ctx context.Context,
	r *resource,
	ko *svcapitypes.Bucket,
) (err error) {
	getAccelerateResponse, err := rm.sdkapi.GetBucketAccelerateConfigurationWithContext(ctx, rm.newGetBucketAcceleratePayload(r))
	if err != nil {
		return err
	}
	ko.Spec.Accelerate = rm.setResourceAccelerate(r, getAccelerateResponse)

	getACLResponse, err := rm.sdkapi.GetBucketAclWithContext(ctx, rm.newGetBucketACLPayload(r))
	if err != nil {
		return err
	}
	rm.setResourceACL(ko, getACLResponse)

	getCORSResponse, err := rm.sdkapi.GetBucketCorsWithContext(ctx, rm.newGetBucketCORSPayload(r))
	if err != nil {
		if awsErr, ok := ackerr.AWSError(err); ok && awsErr.Code() == "NoSuchCORSConfiguration" {
			getCORSResponse = &svcsdk.GetBucketCorsOutput{}
		} else {
			return err
		}
	}
	ko.Spec.CORS = rm.setResourceCORS(r, getCORSResponse)

	getEncryptionResponse, err := rm.sdkapi.GetBucketEncryptionWithContext(ctx, rm.newGetBucketEncryptionPayload(r))
	if err != nil {
		if awsErr, ok := ackerr.AWSError(err); ok && awsErr.Code() == "ServerSideEncryptionConfigurationNotFoundError" {
			getEncryptionResponse = &svcsdk.GetBucketEncryptionOutput{
				ServerSideEncryptionConfiguration: &svcsdk.ServerSideEncryptionConfiguration{},
			}
		} else {
			return err
		}
	}
	ko.Spec.Encryption = rm.setResourceEncryption(r, getEncryptionResponse)

	getLifecycleResponse, err := rm.sdkapi.GetBucketLifecycleConfigurationWithContext(ctx, rm.newGetBucketLifecyclePayload(r))
	if err != nil {
		if awsErr, ok := ackerr.AWSError(err); ok && awsErr.Code() == "NoSuchLifecycleConfiguration" {
			getLifecycleResponse = &svcsdk.GetBucketLifecycleConfigurationOutput{}
		} else {
			return err
		}
	}
	ko.Spec.Lifecycle = rm.setResourceLifecycle(r, getLifecycleResponse)

	getLoggingResponse, err := rm.sdkapi.GetBucketLoggingWithContext(ctx, rm.newGetBucketLoggingPayload(r))
	if err != nil {
		return err
	}
	ko.Spec.Logging = rm.setResourceLogging(r, getLoggingResponse)

	getNotificationResponse, err := rm.sdkapi.GetBucketNotificationConfigurationWithContext(ctx, rm.newGetBucketNotificationPayload(r))
	if err != nil {
		return err
	}
	ko.Spec.Notification = rm.setResourceNotification(r, getNotificationResponse)

	getOwnershipControlsResponse, err := rm.sdkapi.GetBucketOwnershipControlsWithContext(ctx, rm.newGetBucketOwnershipControlsPayload(r))
	if err != nil {
		if awsErr, ok := ackerr.AWSError(err); ok && awsErr.Code() == "OwnershipControlsNotFoundError" {
			getOwnershipControlsResponse = &svcsdk.GetBucketOwnershipControlsOutput{
				OwnershipControls: &svcsdk.OwnershipControls{},
			}
		} else {
			return err
		}
	}
	if getOwnershipControlsResponse.OwnershipControls != nil {
		ko.Spec.OwnershipControls = rm.setResourceOwnershipControls(r, getOwnershipControlsResponse)
	} else {
		ko.Spec.OwnershipControls = nil
	}

	getPolicyResponse, err := rm.sdkapi.GetBucketPolicyWithContext(ctx, rm.newGetBucketPolicyPayload(r))
	if err != nil {
		if awsErr, ok := ackerr.AWSError(err); ok && awsErr.Code() == "NoSuchBucketPolicy" {
			getPolicyResponse = &svcsdk.GetBucketPolicyOutput{}
		} else {
			return err
		}
	}
	ko.Spec.Policy = getPolicyResponse.Policy

	getReplicationResponse, err := rm.sdkapi.GetBucketReplicationWithContext(ctx, rm.newGetBucketReplicationPayload(r))
	if err != nil {
		if awsErr, ok := ackerr.AWSError(err); ok && awsErr.Code() == "ReplicationConfigurationNotFoundError" {
			getReplicationResponse = &svcsdk.GetBucketReplicationOutput{}
		} else {
			return err
		}
	}
	if getReplicationResponse.ReplicationConfiguration != nil {
		ko.Spec.Replication = rm.setResourceReplication(r, getReplicationResponse)
	} else {
		ko.Spec.Replication = nil
	}

	getRequestPaymentResponse, err := rm.sdkapi.GetBucketRequestPaymentWithContext(ctx, rm.newGetBucketRequestPaymentPayload(r))
	if err != nil {
		return nil
	}
	ko.Spec.RequestPayment = rm.setResourceRequestPayment(r, getRequestPaymentResponse)

	getTaggingResponse, err := rm.sdkapi.GetBucketTaggingWithContext(ctx, rm.newGetBucketTaggingPayload(r))
	if err != nil {
		if awsErr, ok := ackerr.AWSError(err); ok && awsErr.Code() == "NoSuchTagSet" {
			getTaggingResponse = &svcsdk.GetBucketTaggingOutput{}
		} else {
			return err
		}
	}
	ko.Spec.Tagging = rm.setResourceTagging(r, getTaggingResponse)

	getVersioningResponse, err := rm.sdkapi.GetBucketVersioningWithContext(ctx, rm.newGetBucketVersioningPayload(r))
	if err != nil {
		return err
	}
	ko.Spec.Versioning = rm.setResourceVersioning(r, getVersioningResponse)

	getWebsiteResponse, err := rm.sdkapi.GetBucketWebsiteWithContext(ctx, rm.newGetBucketWebsitePayload(r))
	if err != nil {
		if awsErr, ok := ackerr.AWSError(err); ok && awsErr.Code() == "NoSuchWebsiteConfiguration" {
			getWebsiteResponse = &svcsdk.GetBucketWebsiteOutput{}
		} else {
			return err
		}
	}
	ko.Spec.Website = rm.setResourceWebsite(r, getWebsiteResponse)
	return nil
}

// customPreCompare ensures that default values of nil-able types are
// appropriately replaced with empty maps or structs depending on the default
// output of the SDK.
func customPreCompare(
	a *resource,
	b *resource,
) {
	if a.ko.Spec.Accelerate == nil && b.ko.Spec.Accelerate != nil {
		a.ko.Spec.Accelerate = &svcapitypes.AccelerateConfiguration{}

		if b.ko.Spec.Accelerate.Status != nil &&
			*b.ko.Spec.Accelerate.Status == DefaultAccelerationStatus {
			a.ko.Spec.Accelerate.Status = &DefaultAccelerationStatus
		}
	}
	if a.ko.Spec.ACL != nil {
		// Don't diff grant headers if a canned ACL has been used
		b.ko.Spec.GrantFullControl = nil
		b.ko.Spec.GrantRead = nil
		b.ko.Spec.GrantReadACP = nil
		b.ko.Spec.GrantWrite = nil
		b.ko.Spec.GrantWriteACP = nil

		// Find the canned ACL from the joined possibility string
		if b.ko.Spec.ACL != nil {
			b.ko.Spec.ACL = matchPossibleCannedACL(*a.ko.Spec.ACL, *b.ko.Spec.ACL)
		}
	} else {
		// If we are sure the grants weren't set from the header strings
		if a.ko.Spec.GrantFullControl == nil &&
			a.ko.Spec.GrantRead == nil &&
			a.ko.Spec.GrantReadACP == nil &&
			a.ko.Spec.GrantWrite == nil &&
			a.ko.Spec.GrantWriteACP == nil {
			b.ko.Spec.GrantFullControl = nil
			b.ko.Spec.GrantRead = nil
			b.ko.Spec.GrantReadACP = nil
			b.ko.Spec.GrantWrite = nil
			b.ko.Spec.GrantWriteACP = nil
		}

		emptyGrant := ""
		if a.ko.Spec.GrantFullControl == nil && b.ko.Spec.GrantFullControl != nil {
			a.ko.Spec.GrantFullControl = &emptyGrant
			// TODO(RedbackThomson): Remove the following line. GrantFullControl
			// has a server-side default of id="<owner ID>". This field needs to
			// be marked as such before we can diff it.
			b.ko.Spec.GrantFullControl = &emptyGrant
		}
		if a.ko.Spec.GrantRead == nil && b.ko.Spec.GrantRead != nil {
			a.ko.Spec.GrantRead = &emptyGrant
		}
		if a.ko.Spec.GrantReadACP == nil && b.ko.Spec.GrantReadACP != nil {
			a.ko.Spec.GrantReadACP = &emptyGrant
		}
		if a.ko.Spec.GrantWrite == nil && b.ko.Spec.GrantWrite != nil {
			a.ko.Spec.GrantWrite = &emptyGrant
		}
		if a.ko.Spec.GrantWriteACP == nil && b.ko.Spec.GrantWriteACP != nil {
			a.ko.Spec.GrantWriteACP = &emptyGrant
		}
	}

	if a.ko.Spec.CORS == nil && b.ko.Spec.CORS != nil {
		a.ko.Spec.CORS = &svcapitypes.CORSConfiguration{}
	}
	if a.ko.Spec.Encryption == nil && b.ko.Spec.Encryption != nil {
		a.ko.Spec.Encryption = &svcapitypes.ServerSideEncryptionConfiguration{}
	}
	if a.ko.Spec.Lifecycle == nil && b.ko.Spec.Lifecycle != nil {
		a.ko.Spec.Lifecycle = &svcapitypes.BucketLifecycleConfiguration{}
	}
	if a.ko.Spec.Logging == nil && b.ko.Spec.Logging != nil {
		a.ko.Spec.Logging = &svcapitypes.BucketLoggingStatus{}
	}
	if a.ko.Spec.Notification == nil && b.ko.Spec.Notification != nil {
		a.ko.Spec.Notification = &svcapitypes.NotificationConfiguration{}
	}
	if a.ko.Spec.OwnershipControls == nil && b.ko.Spec.OwnershipControls != nil {
		a.ko.Spec.OwnershipControls = &svcapitypes.OwnershipControls{}
	}
	if a.ko.Spec.Replication == nil && b.ko.Spec.Replication != nil {
		a.ko.Spec.Replication = &svcapitypes.ReplicationConfiguration{}
	}
	if a.ko.Spec.RequestPayment == nil && b.ko.Spec.RequestPayment != nil {
		a.ko.Spec.RequestPayment = &svcapitypes.RequestPaymentConfiguration{
			Payer: &DefaultRequestPayer,
		}
	}
	if a.ko.Spec.Tagging == nil && b.ko.Spec.Tagging != nil {
		a.ko.Spec.Tagging = &svcapitypes.Tagging{}
	}
	if a.ko.Spec.Versioning == nil && b.ko.Spec.Versioning != nil {
		a.ko.Spec.Versioning = &svcapitypes.VersioningConfiguration{}

		if b.ko.Spec.Versioning.Status != nil &&
			*b.ko.Spec.Versioning.Status == DefaultVersioningStatus {
			a.ko.Spec.Versioning.Status = &DefaultVersioningStatus
		}
	}
	if a.ko.Spec.Website == nil && b.ko.Spec.Website != nil {
		a.ko.Spec.Website = &svcapitypes.WebsiteConfiguration{}
	}
}

//region accelerate

func (rm *resourceManager) newGetBucketAcceleratePayload(
	r *resource,
) *svcsdk.GetBucketAccelerateConfigurationInput {
	res := &svcsdk.GetBucketAccelerateConfigurationInput{}
	res.SetBucket(*r.ko.Spec.Name)
	return res
}

func (rm *resourceManager) newPutBucketAcceleratePayload(
	r *resource,
) *svcsdk.PutBucketAccelerateConfigurationInput {
	res := &svcsdk.PutBucketAccelerateConfigurationInput{}
	res.SetBucket(*r.ko.Spec.Name)
	if r.ko.Spec.Accelerate != nil {
		res.SetAccelerateConfiguration(rm.newAccelerateConfiguration(r))
	} else {
		res.SetAccelerateConfiguration(&svcsdk.AccelerateConfiguration{})
	}

	if res.AccelerateConfiguration.Status == nil {
		res.AccelerateConfiguration.SetStatus(DefaultAccelerationStatus)
	}

	return res
}

func (rm *resourceManager) syncAccelerate(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncAccelerate")
	defer exit(err)
	input := rm.newPutBucketAcceleratePayload(r)

	_, err = rm.sdkapi.PutBucketAccelerateConfigurationWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "PutBucketAccelerate", err)
	if err != nil {
		return err
	}

	return nil
}

//endregion accelerate

//region acl

// setResourceACL sets the `Grant*` spec fields given the output of a
// `GetBucketAcl` operation.
func (rm *resourceManager) setResourceACL(
	ko *svcapitypes.Bucket,
	resp *svcsdk.GetBucketAclOutput,
) {
	grants := GetHeadersFromGrants(resp)
	ko.Spec.GrantFullControl = &grants.FullControl
	ko.Spec.GrantRead = &grants.Read
	ko.Spec.GrantReadACP = &grants.ReadACP
	ko.Spec.GrantWrite = &grants.Write
	ko.Spec.GrantWriteACP = &grants.WriteACP

	// Join possible ACLs into a single string, delimited by bar
	cannedACLs := GetPossibleCannedACLsFromGrants(resp)
	joinedACLs := strings.Join(cannedACLs, CannedACLJoinDelimiter)
	ko.Spec.ACL = &joinedACLs
}

// matchPossibleCannedACL attempts to find a canned ACL string in a joined
// list of possibilities. If any of the possibilities matches, it will be
// returned, otherwise nil.
func matchPossibleCannedACL(search string, joinedPossibilities string) *string {
	splitPossibilities := strings.Split(joinedPossibilities, CannedACLJoinDelimiter)
	for _, possible := range splitPossibilities {
		if search == possible {
			return &possible
		}
	}
	return nil
}

func (rm *resourceManager) newGetBucketACLPayload(
	r *resource,
) *svcsdk.GetBucketAclInput {
	res := &svcsdk.GetBucketAclInput{}
	res.SetBucket(*r.ko.Spec.Name)
	return res
}

func (rm *resourceManager) newPutBucketACLPayload(
	r *resource,
) *svcsdk.PutBucketAclInput {
	res := &svcsdk.PutBucketAclInput{}
	res.SetBucket(*r.ko.Spec.Name)
	if r.ko.Spec.ACL != nil {
		res.SetACL(*r.ko.Spec.ACL)
	}

	if r.ko.Spec.GrantFullControl != nil {
		res.SetGrantFullControl(*r.ko.Spec.GrantFullControl)
	}
	if r.ko.Spec.GrantRead != nil {
		res.SetGrantRead(*r.ko.Spec.GrantRead)
	}
	if r.ko.Spec.GrantReadACP != nil {
		res.SetGrantReadACP(*r.ko.Spec.GrantReadACP)
	}
	if r.ko.Spec.GrantWrite != nil {
		res.SetGrantWrite(*r.ko.Spec.GrantWrite)
	}
	if r.ko.Spec.GrantWriteACP != nil {
		res.SetGrantWriteACP(*r.ko.Spec.GrantWriteACP)
	}

	// Check that there is at least some ACL on the bucket
	if res.ACL == nil &&
		res.GrantFullControl == nil &&
		res.GrantRead == nil &&
		res.GrantReadACP == nil &&
		res.GrantWrite == nil &&
		res.GrantWriteACP == nil {
		res.SetACL(DefaultACL)
	}

	return res
}

func (rm *resourceManager) syncACL(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncACL")
	defer exit(err)
	input := rm.newPutBucketACLPayload(r)

	_, err = rm.sdkapi.PutBucketAclWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "PutBucketAcl", err)
	if err != nil {
		return err
	}

	return nil
}

//endregion acl

//region cors

func (rm *resourceManager) newGetBucketCORSPayload(
	r *resource,
) *svcsdk.GetBucketCorsInput {
	res := &svcsdk.GetBucketCorsInput{}
	res.SetBucket(*r.ko.Spec.Name)
	return res
}

func (rm *resourceManager) newPutBucketCORSPayload(
	r *resource,
) *svcsdk.PutBucketCorsInput {
	res := &svcsdk.PutBucketCorsInput{}
	res.SetBucket(*r.ko.Spec.Name)
	res.SetCORSConfiguration(rm.newCORSConfiguration(r))

	if res.CORSConfiguration.CORSRules == nil {
		res.CORSConfiguration.SetCORSRules([]*svcsdk.CORSRule{})
	}

	return res
}

func (rm *resourceManager) newDeleteBucketCORSPayload(
	r *resource,
) *svcsdk.DeleteBucketCorsInput {
	res := &svcsdk.DeleteBucketCorsInput{}
	res.SetBucket(*r.ko.Spec.Name)

	return res
}

func (rm *resourceManager) putCORS(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.putCORS")
	defer exit(err)
	input := rm.newPutBucketCORSPayload(r)

	_, err = rm.sdkapi.PutBucketCorsWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "PutBucketCors", err)
	if err != nil {
		return err
	}

	return nil
}

func (rm *resourceManager) deleteCORS(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.deleteCORS")
	defer exit(err)
	input := rm.newDeleteBucketCORSPayload(r)

	_, err = rm.sdkapi.DeleteBucketCorsWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "DeleteBucketCors", err)
	if err != nil {
		return err
	}

	return nil
}

func (rm *resourceManager) syncCORS(
	ctx context.Context,
	r *resource,
) (err error) {
	if r.ko.Spec.CORS == nil || r.ko.Spec.CORS.CORSRules == nil {
		return rm.deleteCORS(ctx, r)
	}
	return rm.putCORS(ctx, r)
}

//endregion cors

//region encryption

func (rm *resourceManager) newGetBucketEncryptionPayload(
	r *resource,
) *svcsdk.GetBucketEncryptionInput {
	res := &svcsdk.GetBucketEncryptionInput{}
	res.SetBucket(*r.ko.Spec.Name)
	return res
}

func (rm *resourceManager) newPutBucketEncryptionPayload(
	r *resource,
) *svcsdk.PutBucketEncryptionInput {
	res := &svcsdk.PutBucketEncryptionInput{}
	res.SetBucket(*r.ko.Spec.Name)
	res.SetServerSideEncryptionConfiguration(rm.newServerSideEncryptionConfiguration(r))

	return res
}

func (rm *resourceManager) newDeleteBucketEncryptionPayload(
	r *resource,
) *svcsdk.DeleteBucketEncryptionInput {
	res := &svcsdk.DeleteBucketEncryptionInput{}
	res.SetBucket(*r.ko.Spec.Name)

	return res
}

func (rm *resourceManager) putEncryption(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.putEncryption")
	defer exit(err)
	input := rm.newPutBucketEncryptionPayload(r)

	_, err = rm.sdkapi.PutBucketEncryptionWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "PutBucketEncryption", err)
	if err != nil {
		return err
	}

	return nil
}

func (rm *resourceManager) deleteEncryption(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.deleteEncryption")
	defer exit(err)
	input := rm.newDeleteBucketEncryptionPayload(r)

	_, err = rm.sdkapi.DeleteBucketEncryptionWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "DeleteBucketEncryption", err)
	if err != nil {
		return err
	}

	return nil
}

func (rm *resourceManager) syncEncryption(
	ctx context.Context,
	r *resource,
) (err error) {
	if r.ko.Spec.Encryption == nil || r.ko.Spec.Encryption.Rules == nil {
		return rm.deleteEncryption(ctx, r)
	}
	return rm.putEncryption(ctx, r)
}

//endregion encryption

//region lifecycle

func (rm *resourceManager) newGetBucketLifecyclePayload(
	r *resource,
) *svcsdk.GetBucketLifecycleConfigurationInput {
	res := &svcsdk.GetBucketLifecycleConfigurationInput{}
	res.SetBucket(*r.ko.Spec.Name)
	return res
}

func (rm *resourceManager) newPutBucketLifecyclePayload(
	r *resource,
) *svcsdk.PutBucketLifecycleConfigurationInput {
	res := &svcsdk.PutBucketLifecycleConfigurationInput{}
	res.SetBucket(*r.ko.Spec.Name)
	res.SetLifecycleConfiguration(rm.newLifecycleConfiguration(r))
	return res
}

func (rm *resourceManager) newDeleteBucketLifecyclePayload(
	r *resource,
) *svcsdk.DeleteBucketLifecycleInput {
	res := &svcsdk.DeleteBucketLifecycleInput{}
	res.SetBucket(*r.ko.Spec.Name)
	return res
}

func (rm *resourceManager) putLifecycle(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.putLifecycle")
	defer exit(err)
	input := rm.newPutBucketLifecyclePayload(r)

	_, err = rm.sdkapi.PutBucketLifecycleConfigurationWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "PutBucketLifecycle", err)
	if err != nil {
		return err
	}

	return nil
}

func (rm *resourceManager) deleteLifecycle(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.deleteLifecycle")
	defer exit(err)
	input := rm.newDeleteBucketLifecyclePayload(r)

	_, err = rm.sdkapi.DeleteBucketLifecycleWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "DeleteBucketLifecycle", err)
	if err != nil {
		return err
	}

	return nil
}

func (rm *resourceManager) syncLifecycle(
	ctx context.Context,
	r *resource,
) (err error) {
	if r.ko.Spec.Lifecycle == nil || r.ko.Spec.Lifecycle.Rules == nil {
		return rm.deleteLifecycle(ctx, r)
	}
	return rm.putLifecycle(ctx, r)
}

//endregion lifecycle

//region logging

func (rm *resourceManager) newGetBucketLoggingPayload(
	r *resource,
) *svcsdk.GetBucketLoggingInput {
	res := &svcsdk.GetBucketLoggingInput{}
	res.SetBucket(*r.ko.Spec.Name)
	return res
}

func (rm *resourceManager) newPutBucketLoggingPayload(
	r *resource,
) *svcsdk.PutBucketLoggingInput {
	res := &svcsdk.PutBucketLoggingInput{}
	res.SetBucket(*r.ko.Spec.Name)
	if r.ko.Spec.Logging != nil {
		res.SetBucketLoggingStatus(rm.newBucketLoggingStatus(r))
	} else {
		res.SetBucketLoggingStatus(&svcsdk.BucketLoggingStatus{})
	}
	return res
}

func (rm *resourceManager) syncLogging(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncLogging")
	defer exit(err)
	input := rm.newPutBucketLoggingPayload(r)

	_, err = rm.sdkapi.PutBucketLoggingWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "PutBucketLogging", err)
	if err != nil {
		return err
	}

	return nil
}

//endregion logging

//region notification

func (rm *resourceManager) newGetBucketNotificationPayload(
	r *resource,
) *svcsdk.GetBucketNotificationConfigurationRequest {
	res := &svcsdk.GetBucketNotificationConfigurationRequest{}
	res.SetBucket(*r.ko.Spec.Name)
	return res
}

func (rm *resourceManager) newPutBucketNotificationPayload(
	r *resource,
) *svcsdk.PutBucketNotificationConfigurationInput {
	res := &svcsdk.PutBucketNotificationConfigurationInput{}
	res.SetBucket(*r.ko.Spec.Name)
	if r.ko.Spec.Notification != nil {
		res.SetNotificationConfiguration(rm.newNotificationConfiguration(r))
	} else {
		res.SetNotificationConfiguration(&svcsdk.NotificationConfiguration{})
	}
	return res
}

func (rm *resourceManager) syncNotification(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncNotification")
	defer exit(err)
	input := rm.newPutBucketNotificationPayload(r)

	_, err = rm.sdkapi.PutBucketNotificationConfigurationWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "PutBucketNotification", err)
	if err != nil {
		return err
	}

	return nil
}

//endregion notification

//region ownershipcontrols

func (rm *resourceManager) newGetBucketOwnershipControlsPayload(
	r *resource,
) *svcsdk.GetBucketOwnershipControlsInput {
	res := &svcsdk.GetBucketOwnershipControlsInput{}
	res.SetBucket(*r.ko.Spec.Name)
	return res
}

func (rm *resourceManager) newPutBucketOwnershipControlsPayload(
	r *resource,
) *svcsdk.PutBucketOwnershipControlsInput {
	res := &svcsdk.PutBucketOwnershipControlsInput{}
	res.SetBucket(*r.ko.Spec.Name)
	res.SetOwnershipControls(rm.newOwnershipControls(r))

	return res
}

func (rm *resourceManager) newDeleteBucketOwnershipControlsPayload(
	r *resource,
) *svcsdk.DeleteBucketOwnershipControlsInput {
	res := &svcsdk.DeleteBucketOwnershipControlsInput{}
	res.SetBucket(*r.ko.Spec.Name)

	return res
}

func (rm *resourceManager) putOwnershipControls(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.putOwnershipControls")
	defer exit(err)
	input := rm.newPutBucketOwnershipControlsPayload(r)

	_, err = rm.sdkapi.PutBucketOwnershipControlsWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "PutBucketOwnershipControls", err)
	if err != nil {
		return err
	}

	return nil
}

func (rm *resourceManager) deleteOwnershipControls(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.deleteOwnershipControls")
	defer exit(err)
	input := rm.newDeleteBucketOwnershipControlsPayload(r)

	_, err = rm.sdkapi.DeleteBucketOwnershipControlsWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "DeleteBucketOwnershipControls", err)
	if err != nil {
		return err
	}

	return nil
}

func (rm *resourceManager) syncOwnershipControls(
	ctx context.Context,
	r *resource,
) (err error) {
	if r.ko.Spec.OwnershipControls == nil || r.ko.Spec.OwnershipControls.Rules == nil {
		return rm.deleteOwnershipControls(ctx, r)
	}
	return rm.putOwnershipControls(ctx, r)
}

//endregion ownershipcontrols

//region policy

func (rm *resourceManager) newGetBucketPolicyPayload(
	r *resource,
) *svcsdk.GetBucketPolicyInput {
	res := &svcsdk.GetBucketPolicyInput{}
	res.SetBucket(*r.ko.Spec.Name)
	return res
}

func (rm *resourceManager) newPutBucketPolicyPayload(
	r *resource,
) *svcsdk.PutBucketPolicyInput {
	res := &svcsdk.PutBucketPolicyInput{}
	res.SetBucket(*r.ko.Spec.Name)
	res.SetConfirmRemoveSelfBucketAccess(false)
	res.SetPolicy(*r.ko.Spec.Policy)

	return res
}

func (rm *resourceManager) newDeleteBucketPolicyPayload(
	r *resource,
) *svcsdk.DeleteBucketPolicyInput {
	res := &svcsdk.DeleteBucketPolicyInput{}
	res.SetBucket(*r.ko.Spec.Name)
	return res
}

func (rm *resourceManager) putPolicy(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.putPolicy")
	defer exit(err)
	input := rm.newPutBucketPolicyPayload(r)

	_, err = rm.sdkapi.PutBucketPolicyWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "PutBucketPolicy", err)
	if err != nil {
		return err
	}

	return nil
}

func (rm *resourceManager) deletePolicy(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.deletePolicy")
	defer exit(err)
	input := rm.newDeleteBucketPolicyPayload(r)

	_, err = rm.sdkapi.DeleteBucketPolicyWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "DeleteBucketPolicy", err)
	if err != nil {
		return err
	}

	return nil
}

func (rm *resourceManager) syncPolicy(
	ctx context.Context,
	r *resource,
) (err error) {
	if r.ko.Spec.Policy == nil {
		return rm.deletePolicy(ctx, r)
	}
	return rm.putPolicy(ctx, r)
}

//endregion

//region replication

func (rm *resourceManager) newGetBucketReplicationPayload(
	r *resource,
) *svcsdk.GetBucketReplicationInput {
	res := &svcsdk.GetBucketReplicationInput{}
	res.SetBucket(*r.ko.Spec.Name)
	return res
}

func (rm *resourceManager) newPutBucketReplicationPayload(
	r *resource,
) *svcsdk.PutBucketReplicationInput {
	res := &svcsdk.PutBucketReplicationInput{}
	res.SetBucket(*r.ko.Spec.Name)
	res.SetReplicationConfiguration(rm.newReplicationConfiguration(r))
	return res
}

func (rm *resourceManager) newDeleteBucketReplicationPayload(
	r *resource,
) *svcsdk.DeleteBucketReplicationInput {
	res := &svcsdk.DeleteBucketReplicationInput{}
	res.SetBucket(*r.ko.Spec.Name)
	return res
}

func (rm *resourceManager) putReplication(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.putReplication")
	defer exit(err)
	input := rm.newPutBucketReplicationPayload(r)

	_, err = rm.sdkapi.PutBucketReplicationWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "PutBucketReplication", err)
	if err != nil {
		return err
	}

	return nil
}

func (rm *resourceManager) deleteReplication(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.deleteReplication")
	defer exit(err)
	input := rm.newDeleteBucketReplicationPayload(r)

	_, err = rm.sdkapi.DeleteBucketReplicationWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "DeleteBucketReplication", err)
	if err != nil {
		return err
	}

	return nil
}

func (rm *resourceManager) syncReplication(
	ctx context.Context,
	r *resource,
) (err error) {
	if r.ko.Spec.Replication == nil || r.ko.Spec.Replication.Rules == nil {
		return rm.deleteReplication(ctx, r)
	}
	return rm.putReplication(ctx, r)
}

//endregion replication

//region requestpayment

func (rm *resourceManager) newGetBucketRequestPaymentPayload(
	r *resource,
) *svcsdk.GetBucketRequestPaymentInput {
	res := &svcsdk.GetBucketRequestPaymentInput{}
	res.SetBucket(*r.ko.Spec.Name)
	return res
}

func (rm *resourceManager) newPutBucketRequestPaymentPayload(
	r *resource,
) *svcsdk.PutBucketRequestPaymentInput {
	res := &svcsdk.PutBucketRequestPaymentInput{}
	res.SetBucket(*r.ko.Spec.Name)
	if r.ko.Spec.RequestPayment != nil && r.ko.Spec.RequestPayment.Payer != nil {
		res.SetRequestPaymentConfiguration(rm.newRequestPaymentConfiguration(r))
	} else {
		res.SetRequestPaymentConfiguration(&svcsdk.RequestPaymentConfiguration{})
	}

	if res.RequestPaymentConfiguration.Payer == nil {
		res.RequestPaymentConfiguration.SetPayer(DefaultRequestPayer)
	}

	return res
}

func (rm *resourceManager) syncRequestPayment(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncRequestPayment")
	defer exit(err)
	input := rm.newPutBucketRequestPaymentPayload(r)

	_, err = rm.sdkapi.PutBucketRequestPaymentWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "PutBucketRequestPayment", err)
	if err != nil {
		return err
	}

	return nil
}

//endregion requestpayment

//region tagging

func (rm *resourceManager) newGetBucketTaggingPayload(
	r *resource,
) *svcsdk.GetBucketTaggingInput {
	res := &svcsdk.GetBucketTaggingInput{}
	res.SetBucket(*r.ko.Spec.Name)
	return res
}

func (rm *resourceManager) newPutBucketTaggingPayload(
	r *resource,
) *svcsdk.PutBucketTaggingInput {
	res := &svcsdk.PutBucketTaggingInput{}
	res.SetBucket(*r.ko.Spec.Name)
	res.SetTagging(rm.newTagging(r))

	return res
}

func (rm *resourceManager) newDeleteBucketTaggingPayload(
	r *resource,
) *svcsdk.DeleteBucketTaggingInput {
	res := &svcsdk.DeleteBucketTaggingInput{}
	res.SetBucket(*r.ko.Spec.Name)

	return res
}

func (rm *resourceManager) putTagging(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.putTagging")
	defer exit(err)
	input := rm.newPutBucketTaggingPayload(r)

	_, err = rm.sdkapi.PutBucketTaggingWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "PutBucketTagging", err)
	if err != nil {
		return err
	}

	return nil
}

func (rm *resourceManager) deleteTagging(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.deleteTagging")
	defer exit(err)
	input := rm.newDeleteBucketTaggingPayload(r)

	_, err = rm.sdkapi.DeleteBucketTaggingWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "DeleteBucketTagging", err)
	if err != nil {
		return err
	}

	return nil
}

func (rm *resourceManager) syncTagging(
	ctx context.Context,
	r *resource,
) (err error) {
	if r.ko.Spec.Tagging == nil || r.ko.Spec.Tagging.TagSet == nil {
		return rm.deleteTagging(ctx, r)
	}
	return rm.putTagging(ctx, r)
}

//endregion tagging

//region versioning

func (rm *resourceManager) newGetBucketVersioningPayload(
	r *resource,
) *svcsdk.GetBucketVersioningInput {
	res := &svcsdk.GetBucketVersioningInput{}
	res.SetBucket(*r.ko.Spec.Name)
	return res
}

func (rm *resourceManager) newPutBucketVersioningPayload(
	r *resource,
) *svcsdk.PutBucketVersioningInput {
	res := &svcsdk.PutBucketVersioningInput{}
	res.SetBucket(*r.ko.Spec.Name)
	if r.ko.Spec.Versioning != nil {
		res.SetVersioningConfiguration(rm.newVersioningConfiguration(r))
	} else {
		res.SetVersioningConfiguration(&svcsdk.VersioningConfiguration{})
	}

	if res.VersioningConfiguration.Status == nil {
		res.VersioningConfiguration.SetStatus(DefaultVersioningStatus)
	}

	return res
}

func (rm *resourceManager) syncVersioning(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncVersioning")
	defer exit(err)
	input := rm.newPutBucketVersioningPayload(r)

	_, err = rm.sdkapi.PutBucketVersioningWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "PutBucketVersioning", err)
	if err != nil {
		return err
	}

	return nil
}

//endregion versioning

//region website

func (rm *resourceManager) newGetBucketWebsitePayload(
	r *resource,
) *svcsdk.GetBucketWebsiteInput {
	res := &svcsdk.GetBucketWebsiteInput{}
	res.SetBucket(*r.ko.Spec.Name)
	return res
}

func (rm *resourceManager) newPutBucketWebsitePayload(
	r *resource,
) *svcsdk.PutBucketWebsiteInput {
	res := &svcsdk.PutBucketWebsiteInput{}
	res.SetBucket(*r.ko.Spec.Name)
	res.SetWebsiteConfiguration(rm.newWebsiteConfiguration(r))

	return res
}

func (rm *resourceManager) newDeleteBucketWebsitePayload(
	r *resource,
) *svcsdk.DeleteBucketWebsiteInput {
	res := &svcsdk.DeleteBucketWebsiteInput{}
	res.SetBucket(*r.ko.Spec.Name)

	return res
}

func (rm *resourceManager) putWebsite(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.putWebsite")
	defer exit(err)
	input := rm.newPutBucketWebsitePayload(r)

	_, err = rm.sdkapi.PutBucketWebsiteWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "PutBucketWebsite", err)
	if err != nil {
		return err
	}

	return nil
}

func (rm *resourceManager) deleteWebsite(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.deleteWebsite")
	defer exit(err)
	input := rm.newDeleteBucketWebsitePayload(r)

	_, err = rm.sdkapi.DeleteBucketWebsiteWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "DeleteBucketWebsite", err)
	if err != nil {
		return err
	}

	return nil
}

func (rm *resourceManager) syncWebsite(
	ctx context.Context,
	r *resource,
) (err error) {
	if r.ko.Spec.Website == nil {
		return rm.deleteWebsite(ctx, r)
	}
	return rm.putWebsite(ctx, r)
}

//endregion website
