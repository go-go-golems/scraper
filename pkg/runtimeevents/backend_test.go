package runtimeevents

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenPublisherSubscriberGoChannel(t *testing.T) {
	resources, err := OpenPublisherSubscriber(Config{Backend: BackendGoChannel})
	require.NoError(t, err)
	defer func() { require.NoError(t, resources.Close()) }()

	require.Equal(t, DefaultTopic, resources.Topic)
	require.NotNil(t, resources.Publisher)
	require.NotNil(t, resources.Subscriber)
}

func TestOpenPublisherOffBackendIsNoop(t *testing.T) {
	resources, err := OpenPublisher(Config{})
	require.NoError(t, err)
	defer func() { require.NoError(t, resources.Close()) }()

	require.Equal(t, DefaultTopic, resources.Topic)
	require.NotNil(t, resources.Publisher)
	require.Nil(t, resources.Subscriber)
	require.NoError(t, resources.Publisher.Publish(resources.Topic))
}

func TestConfigValidateRejectsUnknownBackend(t *testing.T) {
	err := Config{Backend: Backend("bogus")}.Validate()
	require.Error(t, err)
}
