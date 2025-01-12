package cri

import (
	"context"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	client "github.com/weaveworks/scope/cri/runtime"
	"github.com/weaveworks/scope/probe/docker"
	"github.com/weaveworks/scope/report"
)

// Reporter generate Reports containing Container and ContainerImage topologies
type Reporter struct {
	cri            client.RuntimeServiceClient
	criImageClient client.ImageServiceClient
}

// NewReporter makes a new Reporter
func NewReporter(cri client.RuntimeServiceClient, criImageClient client.ImageServiceClient) *Reporter {
	reporter := &Reporter{
		cri:            cri,
		criImageClient: criImageClient,
	}

	return reporter
}

// Name of this reporter, for metrics gathering
func (Reporter) Name() string { return "CRI" }

// Report generates a Report containing Container topologies
func (r *Reporter) Report() (report.Report, error) {
	result := report.MakeReport()
	containerTopol, err := r.containerTopology()
	if err != nil {
		return report.MakeReport(), err
	}

	imageTopol, err := r.containerImageTopology()
	if err != nil {
		return report.MakeReport(), err
	}

	result.Container = result.Container.Merge(containerTopol)
	result.ContainerImage = result.ContainerImage.Merge(imageTopol)
	return result, nil
}

func (r *Reporter) containerTopology() (report.Topology, error) {
	result := report.MakeTopology().
		WithMetadataTemplates(docker.ContainerImageMetadataTemplates).
		WithTableTemplates(docker.ContainerImageTableTemplates)

	ctx := context.Background()
	resp, err := r.cri.ListContainers(ctx, &client.ListContainersRequest{})
	if err != nil {
		return result, err
	}

	for _, c := range resp.Containers {
		result.AddNode(getNode(c))
	}

	return result, nil
}

func getNode(c *client.Container) report.Node {
	result := report.MakeNodeWith(report.MakeContainerNodeID(c.Id), map[string]string{
		docker.ContainerName:       c.Metadata.Name,
		docker.ContainerID:         c.Id,
		docker.ContainerState:      getState(c),
		docker.ContainerStateHuman: getState(c),
		//docker.ContainerRestartCount: fmt.Sprintf("%v", c.Metadata.Attempt),
		docker.ImageID: trimImageID(c.ImageRef),
	}).WithParents(report.MakeSets().
		Add(report.ContainerImage, report.MakeStringSet(report.MakeContainerImageNodeID(c.ImageRef))),
	)
	result = result.AddPrefixPropertyList(docker.LabelPrefix, c.Labels)
	return result
}

func getState(c *client.Container) string {
	switch c.State.String() {
	case "CONTAINER_RUNNING":
		return report.StateRunning
	case "CONTAINER_EXITED":
		return report.StateExited
	case "CONTAINER_UNKNOWN":
		return report.StateUnknown
	case "CONTAINER_CREATED":
		return report.StateCreated
	default:
		return report.StateUnknown
	}
}

func (r *Reporter) containerImageTopology() (report.Topology, error) {
	result := report.MakeTopology().
		WithMetadataTemplates(docker.ContainerImageMetadataTemplates).
		WithTableTemplates(docker.ContainerImageTableTemplates)

	ctx := context.Background()
	resp, err := r.criImageClient.ListImages(ctx, &client.ListImagesRequest{})
	if err != nil {
		return result, err
	}

	for _, img := range resp.Images {
		result.AddNode(getImage(img))
	}

	return result, nil
}

func getImage(image *client.Image) report.Node {
	// logrus.Infof("images: %v", image)
	// image format: sha256:ab21abc2d2c34c2b2d2c23bbcf23gg23f23
	imageID := trimImageID(image.Id)
	latests := map[string]string{
		docker.ImageID:        imageID,
		docker.ImageSize:      humanize.Bytes(uint64(image.Size())),
		docker.ImageCreatedAt: time.Unix(0, 0).Format("2006-01-02T15:04:05") + "Z",
	}
	if len(image.RepoTags) > 0 {
		imageFullName := image.RepoTags[0]
		latests[docker.ImageName] = docker.ImageNameWithoutTag(imageFullName)
		latests[docker.ImageTag] = docker.ImageNameTag(imageFullName)
	}
	result := report.MakeNodeWith(report.MakeContainerImageNodeID(imageID), latests).WithParents(report.MakeSets().
		Add(report.ContainerImage, report.MakeStringSet(report.MakeContainerImageNodeID(imageID))),
	)
	// todo: remove if useless
	result = result.AddPrefixPropertyList(docker.LabelPrefix, nil)
	return result
}

// CRI sometimes prefixes ids with a "type" annotation, but it renders a bit
// ugly and isn't necessary, so we should strip it off
func trimImageID(id string) string {
	return strings.TrimPrefix(id, "sha256:")
}
