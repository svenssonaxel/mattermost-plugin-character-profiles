package main_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mattermost/mattermost-server/v5/model"

	"github.com/svenssonaxel/mattermost-plugin-character-profiles/server"
)

func TestDeepClonePost(t *testing.T) {
	// Initialize posts
	post := &model.Post{
		Id:         "id",
		ChannelId:  "channelId",
		RootId:     "rootId",
		ParentId:   "parentId",
		OriginalId: "originalId",
		Message:    "message",
		Type:       "type",
		Props: model.StringInterface{
			"key": "value",
		},
	}
	postJson1 := post.ToJson()
	clone1 := main.DeepClonePost(post)
	clone1Json1 := clone1.ToJson()
	clone2 := main.DeepClonePost(post)
	clone2Json1 := clone2.ToJson()
	// Change clone1
	clone1.Message = "new message"
	clone1.AddProp("key", "new value")
	clone1.AddProp("newKey", "new value2")
	// Capture results
	postJson2 := post.ToJson()
	clone1Json2 := clone1.ToJson()
	clone2Json2 := clone2.ToJson()
	assert.Equal(t, postJson1, postJson2)
	assert.Equal(t, postJson1, clone1Json1)
	assert.NotEqual(t, postJson1, clone1Json2)
	assert.Equal(t, postJson1, clone2Json1)
	assert.Equal(t, postJson1, clone2Json2)
}
