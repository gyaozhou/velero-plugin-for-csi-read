/*
Copyright 2020 the Velero contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package backup

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	snapshotv1api "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/vmware-tanzu/velero-plugin-for-csi/internal/util"
	velerov1api "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	biav2 "github.com/vmware-tanzu/velero/pkg/plugin/velero/backupitemaction/v2"
)

// VolumeSnapshotClassBackupItemAction is a backup item action plugin to backup
// CSI VolumeSnapshotclass objects using Velero
type VolumeSnapshotClassBackupItemAction struct {
	Log logrus.FieldLogger
}

// AppliesTo returns information indicating that the VolumeSnapshotClassBackupItemAction action should be invoked to backup volumesnapshotclass.
func (p *VolumeSnapshotClassBackupItemAction) AppliesTo() (velero.ResourceSelector, error) {
	p.Log.Debug("VolumeSnapshotClassBackupItemAction AppliesTo")

	return velero.ResourceSelector{
		IncludedResources: []string{"volumesnapshotclass.snapshot.storage.k8s.io"},
	}, nil
}

// Execute backs up a VolumeSnapshotClass object and returns as additional items any snapshot lister secret that may be referenced in its annotations.
func (p *VolumeSnapshotClassBackupItemAction) Execute(item runtime.Unstructured, backup *velerov1api.Backup) (runtime.Unstructured, []velero.ResourceIdentifier, string, []velero.ResourceIdentifier, error) {
	p.Log.Infof("Executing VolumeSnapshotClassBackupItemAction")

	var snapClass snapshotv1api.VolumeSnapshotClass
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.UnstructuredContent(), &snapClass); err != nil {
		return nil, nil, "", nil, errors.WithStack(err)
	}

	// zhou: README, secrets used by VolumeSnapshotClass ?
	additionalItems := []velero.ResourceIdentifier{}
	if util.IsVolumeSnapshotClassHasListerSecret(&snapClass) {
		additionalItems = append(additionalItems, velero.ResourceIdentifier{
			GroupResource: schema.GroupResource{Group: "", Resource: "secrets"},
			Name:          snapClass.Annotations[util.PrefixedSnapshotterListSecretNameKey],
			Namespace:     snapClass.Annotations[util.PrefixedSnapshotterListSecretNamespaceKey],
		})

		util.AddAnnotations(&snapClass.ObjectMeta, map[string]string{
			util.MustIncludeAdditionalItemAnnotation: "true",
		})
	}

	snapClassMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&snapClass)
	if err != nil {
		return nil, nil, "", nil, errors.WithStack(err)
	}

	p.Log.Infof("Returning from VolumeSnapshotClassBackupItemAction with %d additionalItems to backup", len(additionalItems))
	return &unstructured.Unstructured{Object: snapClassMap}, additionalItems, "", nil, nil
}

func (p *VolumeSnapshotClassBackupItemAction) Name() string {
	return "VolumeSnapshotClassBackupItemAction"
}

func (p *VolumeSnapshotClassBackupItemAction) Progress(operationID string, backup *velerov1api.Backup) (velero.OperationProgress, error) {
	progress := velero.OperationProgress{}
	if operationID == "" {
		return progress, biav2.InvalidOperationIDError(operationID)
	}

	return progress, nil
}

func (p *VolumeSnapshotClassBackupItemAction) Cancel(operationID string, backup *velerov1api.Backup) error {
	return nil
}
