// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost/server/public/model"
)

// Test for MM-13598 where an invalid integration URL was causing a crash
func TestPostActionInvalidURL(t *testing.T) {
	mainHelper.Parallel(t)
	th := Setup(t).InitBasic()
	defer th.TearDown()

	th.App.UpdateConfig(func(cfg *model.Config) {
		*cfg.ServiceSettings.AllowedUntrustedInternalConnections = "localhost,127.0.0.1"
	})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request model.PostActionIntegrationRequest
		jsonErr := json.NewDecoder(r.Body).Decode(&request)
		assert.NoError(t, jsonErr)
	}))
	defer ts.Close()

	interactivePost := model.Post{
		Message:       "Interactive post",
		ChannelId:     th.BasicChannel.Id,
		PendingPostId: model.NewId() + ":" + fmt.Sprint(model.GetMillis()),
		UserId:        th.BasicUser.Id,
		Props: model.StringInterface{
			model.PostPropsAttachments: []*model.SlackAttachment{
				{
					Text: "hello",
					Actions: []*model.PostAction{
						{
							Type: model.PostActionTypeButton,
							Name: "action",
							Integration: &model.PostActionIntegration{
								URL: ":test",
							},
						},
					},
				},
			},
		},
	}

	post, err := th.App.CreatePostAsUser(th.Context, &interactivePost, "", true)
	require.Nil(t, err)
	attachments, ok := post.GetProp(model.PostPropsAttachments).([]*model.SlackAttachment)
	require.True(t, ok)
	require.NotEmpty(t, attachments[0].Actions)
	require.NotEmpty(t, attachments[0].Actions[0].Id)

	_, err = th.App.DoPostActionWithCookie(th.Context, post.Id, attachments[0].Actions[0].Id, th.BasicUser.Id, "", nil)
	require.NotNil(t, err)
	assert.ErrorContains(t, err, "missing protocol scheme")
}

func TestPostActionEmptyResponse(t *testing.T) {
	mainHelper.Parallel(t)
	th := Setup(t).InitBasic()
	defer th.TearDown()

	channel := th.BasicChannel
	th.App.UpdateConfig(func(cfg *model.Config) {
		*cfg.ServiceSettings.AllowedUntrustedInternalConnections = "localhost,127.0.0.1"
	})

	t.Run("Empty response on post action", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		defer ts.Close()

		interactivePost := model.Post{
			Message:       "Interactive post",
			ChannelId:     channel.Id,
			PendingPostId: model.NewId() + ":" + fmt.Sprint(model.GetMillis()),
			UserId:        th.BasicUser.Id,
			Props: model.StringInterface{
				model.PostPropsAttachments: []*model.SlackAttachment{
					{
						Text: "hello",
						Actions: []*model.PostAction{
							{
								Type:       model.PostActionTypeSelect,
								Name:       "action",
								DataSource: model.PostActionDataSourceUsers,
								Integration: &model.PostActionIntegration{
									Context: model.StringInterface{
										"s": "foo",
										"n": 3,
									},
									URL: ts.URL,
								},
							},
						},
					},
				},
			},
		}

		post, err := th.App.CreatePostAsUser(th.Context, &interactivePost, "", true)
		require.Nil(t, err)

		attachments, ok := post.GetProp(model.PostPropsAttachments).([]*model.SlackAttachment)
		require.True(t, ok)

		_, err = th.App.DoPostActionWithCookie(th.Context, post.Id, attachments[0].Actions[0].Id, th.BasicUser.Id, "", nil)
		require.Nil(t, err)
	})

	t.Run("Empty response on post action, timeout", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * time.Second)
		}))
		defer ts.Close()

		interactivePost := model.Post{
			Message:       "Interactive post",
			ChannelId:     channel.Id,
			PendingPostId: model.NewId() + ":" + fmt.Sprint(model.GetMillis()),
			UserId:        th.BasicUser.Id,
			Props: model.StringInterface{
				model.PostPropsAttachments: []*model.SlackAttachment{
					{
						Text: "hello",
						Actions: []*model.PostAction{
							{
								Type:       model.PostActionTypeSelect,
								Name:       "action",
								DataSource: model.PostActionDataSourceUsers,
								Integration: &model.PostActionIntegration{
									Context: model.StringInterface{
										"s": "foo",
										"n": 3,
									},
									URL: ts.URL,
								},
							},
						},
					},
				},
			},
		}

		post, err := th.App.CreatePostAsUser(th.Context, &interactivePost, "", true)
		require.Nil(t, err)

		attachments, ok := post.GetProp(model.PostPropsAttachments).([]*model.SlackAttachment)
		require.True(t, ok)

		th.App.UpdateConfig(func(cfg *model.Config) {
			cfg.ServiceSettings.OutgoingIntegrationRequestsTimeout = model.NewPointer(int64(1))
		})

		_, err = th.App.DoPostActionWithCookie(th.Context, post.Id, attachments[0].Actions[0].Id, th.BasicUser.Id, "", nil)
		require.NotNil(t, err)
		assert.ErrorContains(t, err, "context deadline exceeded")
	})
}

func TestPostAction(t *testing.T) {
	mainHelper.Parallel(t)
	testCases := []struct {
		Description string
		Channel     func(th *TestHelper) *model.Channel
	}{
		{"public channel", func(th *TestHelper) *model.Channel {
			return th.BasicChannel
		}},
		{"direct channel", func(th *TestHelper) *model.Channel {
			user1 := th.CreateUser()

			return th.CreateDmChannel(user1)
		}},
		{"group channel", func(th *TestHelper) *model.Channel {
			user1 := th.CreateUser()
			user2 := th.CreateUser()

			return th.CreateGroupChannel(th.Context, user1, user2)
		}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Description, func(t *testing.T) {
			th := Setup(t).InitBasic()
			defer th.TearDown()

			channel := testCase.Channel(th)

			th.App.UpdateConfig(func(cfg *model.Config) {
				*cfg.ServiceSettings.AllowedUntrustedInternalConnections = "localhost,127.0.0.1"
			})

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request model.PostActionIntegrationRequest
				jsonErr := json.NewDecoder(r.Body).Decode(&request)
				assert.NoError(t, jsonErr)

				assert.Equal(t, th.BasicUser.Id, request.UserId)
				assert.Equal(t, th.BasicUser.Username, request.UserName)
				assert.Equal(t, channel.Id, request.ChannelId)
				assert.Equal(t, channel.Name, request.ChannelName)
				if channel.Type == model.ChannelTypeDirect || channel.Type == model.ChannelTypeGroup {
					assert.Empty(t, request.TeamId)
					assert.Empty(t, request.TeamName)
				} else {
					assert.Equal(t, th.BasicTeam.Id, request.TeamId)
					assert.Equal(t, th.BasicTeam.Name, request.TeamName)
				}
				assert.True(t, request.TriggerId != "")
				if request.Type == model.PostActionTypeSelect {
					if selectedOption, ok := request.Context["selected_option"]; ok {
						// If something was selected, confirm that the data source and selected option are present
						assert.Equal(t, model.PostActionDataSourceUsers, request.DataSource)
						assert.Equal(t, "selected", selectedOption)
					} else {
						assert.Empty(t, request.DataSource)
					}
				} else {
					assert.Equal(t, "", request.DataSource)
				}
				assert.Equal(t, "foo", request.Context["s"])
				assert.EqualValues(t, 3, request.Context["n"])
				fmt.Fprintf(w, `{"post": {"message": "updated"}, "ephemeral_text": "foo"}`)
			}))
			defer ts.Close()

			interactivePost := model.Post{
				Message:       "Interactive post",
				ChannelId:     channel.Id,
				PendingPostId: model.NewId() + ":" + fmt.Sprint(model.GetMillis()),
				UserId:        th.BasicUser.Id,
				Props: model.StringInterface{
					model.PostPropsAttachments: []*model.SlackAttachment{
						{
							Text: "hello",
							Actions: []*model.PostAction{
								{
									Type:       model.PostActionTypeSelect,
									Name:       "action",
									DataSource: model.PostActionDataSourceUsers,
									Integration: &model.PostActionIntegration{
										Context: model.StringInterface{
											"s": "foo",
											"n": 3,
										},
										URL: ts.URL,
									},
								},
							},
						},
					},
				},
			}

			post, err := th.App.CreatePostAsUser(th.Context, &interactivePost, "", true)
			require.Nil(t, err)

			attachments, ok := post.GetProp(model.PostPropsAttachments).([]*model.SlackAttachment)
			require.True(t, ok)

			require.NotEmpty(t, attachments[0].Actions)
			require.NotEmpty(t, attachments[0].Actions[0].Id)

			menuPost := model.Post{
				Message:       "Interactive post",
				ChannelId:     channel.Id,
				PendingPostId: model.NewId() + ":" + fmt.Sprint(model.GetMillis()),
				UserId:        th.BasicUser.Id,
				Props: model.StringInterface{
					model.PostPropsAttachments: []*model.SlackAttachment{
						{
							Text: "hello",
							Actions: []*model.PostAction{
								{
									Type:       model.PostActionTypeSelect,
									Name:       "action",
									DataSource: model.PostActionDataSourceUsers,
									Integration: &model.PostActionIntegration{
										Context: model.StringInterface{
											"s": "foo",
											"n": 3,
										},
										URL: ts.URL,
									},
								},
							},
						},
					},
				},
			}

			post2, err := th.App.CreatePostAsUser(th.Context, &menuPost, "", true)
			require.Nil(t, err)

			attachments2, ok := post2.GetProp(model.PostPropsAttachments).([]*model.SlackAttachment)
			require.True(t, ok)

			require.NotEmpty(t, attachments2[0].Actions)
			require.NotEmpty(t, attachments2[0].Actions[0].Id)

			clientTriggerID, err := th.App.DoPostActionWithCookie(th.Context, post.Id, "notavalidid", th.BasicUser.Id, "", nil)
			require.NotNil(t, err)
			assert.Equal(t, http.StatusNotFound, err.StatusCode)
			assert.Len(t, clientTriggerID, 0)

			clientTriggerID, err = th.App.DoPostActionWithCookie(th.Context, post.Id, attachments[0].Actions[0].Id, th.BasicUser.Id, "", nil)
			require.Nil(t, err)
			assert.Len(t, clientTriggerID, 26)

			clientTriggerID, err = th.App.DoPostActionWithCookie(th.Context, post2.Id, attachments2[0].Actions[0].Id, th.BasicUser.Id, "selected", nil)
			require.Nil(t, err)
			assert.Len(t, clientTriggerID, 26)

			th.App.UpdateConfig(func(cfg *model.Config) {
				*cfg.ServiceSettings.AllowedUntrustedInternalConnections = ""
			})

			_, err = th.App.DoPostActionWithCookie(th.Context, post.Id, attachments[0].Actions[0].Id, th.BasicUser.Id, "", nil)
			require.NotNil(t, err)
			assert.ErrorContains(t, err, "address forbidden")

			interactivePostPlugin := model.Post{
				Message:       "Interactive post",
				ChannelId:     channel.Id,
				PendingPostId: model.NewId() + ":" + fmt.Sprint(model.GetMillis()),
				UserId:        th.BasicUser.Id,
				Props: model.StringInterface{
					model.PostPropsAttachments: []*model.SlackAttachment{
						{
							Text: "hello",
							Actions: []*model.PostAction{
								{
									Type:       model.PostActionTypeSelect,
									Name:       "action",
									DataSource: model.PostActionDataSourceUsers,
									Integration: &model.PostActionIntegration{
										Context: model.StringInterface{
											"s": "foo",
											"n": 3,
										},
										URL: ts.URL + "/plugins/myplugin/myaction",
									},
								},
							},
						},
					},
				},
			}

			postplugin, err := th.App.CreatePostAsUser(th.Context, &interactivePostPlugin, "", true)
			require.Nil(t, err)

			attachmentsPlugin, ok := postplugin.GetProp(model.PostPropsAttachments).([]*model.SlackAttachment)
			require.True(t, ok)

			_, err = th.App.DoPostActionWithCookie(th.Context, postplugin.Id, attachmentsPlugin[0].Actions[0].Id, th.BasicUser.Id, "", nil)
			require.Equal(t, "api.post.do_action.action_integration.app_error", err.Id)

			th.App.UpdateConfig(func(cfg *model.Config) {
				*cfg.ServiceSettings.AllowedUntrustedInternalConnections = "localhost,127.0.0.1"
			})

			_, err = th.App.DoPostActionWithCookie(th.Context, postplugin.Id, attachmentsPlugin[0].Actions[0].Id, th.BasicUser.Id, "", nil)
			require.Nil(t, err)

			th.App.UpdateConfig(func(cfg *model.Config) {
				*cfg.ServiceSettings.SiteURL = "http://127.1.1.1"
			})

			interactivePostSiteURL := model.Post{
				Message:       "Interactive post",
				ChannelId:     channel.Id,
				PendingPostId: model.NewId() + ":" + fmt.Sprint(model.GetMillis()),
				UserId:        th.BasicUser.Id,
				Props: model.StringInterface{
					model.PostPropsAttachments: []*model.SlackAttachment{
						{
							Text: "hello",
							Actions: []*model.PostAction{
								{
									Type:       model.PostActionTypeSelect,
									Name:       "action",
									DataSource: model.PostActionDataSourceUsers,
									Integration: &model.PostActionIntegration{
										Context: model.StringInterface{
											"s": "foo",
											"n": 3,
										},
										URL: "http://127.1.1.1/plugins/myplugin/myaction",
									},
								},
							},
						},
					},
				},
			}

			postSiteURL, err := th.App.CreatePostAsUser(th.Context, &interactivePostSiteURL, "", true)
			require.Nil(t, err)

			attachmentsSiteURL, ok := postSiteURL.GetProp(model.PostPropsAttachments).([]*model.SlackAttachment)
			require.True(t, ok)

			_, err = th.App.DoPostActionWithCookie(th.Context, postSiteURL.Id, attachmentsSiteURL[0].Actions[0].Id, th.BasicUser.Id, "", nil)
			require.NotNil(t, err)
			assert.ErrorContains(t, err, "connection refused")

			th.App.UpdateConfig(func(cfg *model.Config) {
				*cfg.ServiceSettings.SiteURL = ts.URL + "/subpath"
			})

			interactivePostSubpath := model.Post{
				Message:       "Interactive post",
				ChannelId:     channel.Id,
				PendingPostId: model.NewId() + ":" + fmt.Sprint(model.GetMillis()),
				UserId:        th.BasicUser.Id,
				Props: model.StringInterface{
					model.PostPropsAttachments: []*model.SlackAttachment{
						{
							Text: "hello",
							Actions: []*model.PostAction{
								{
									Type:       model.PostActionTypeSelect,
									Name:       "action",
									DataSource: model.PostActionDataSourceUsers,
									Integration: &model.PostActionIntegration{
										Context: model.StringInterface{
											"s": "foo",
											"n": 3,
										},
										URL: ts.URL + "/subpath/plugins/myplugin/myaction",
									},
								},
							},
						},
					},
				},
			}

			postSubpath, err := th.App.CreatePostAsUser(th.Context, &interactivePostSubpath, "", true)
			require.Nil(t, err)

			attachmentsSubpath, ok := postSubpath.GetProp(model.PostPropsAttachments).([]*model.SlackAttachment)
			require.True(t, ok)

			_, err = th.App.DoPostActionWithCookie(th.Context, postSubpath.Id, attachmentsSubpath[0].Actions[0].Id, th.BasicUser.Id, "", nil)
			require.Nil(t, err)
		})
	}
}

func TestPostActionProps(t *testing.T) {
	mainHelper.Parallel(t)
	th := Setup(t).InitBasic()
	defer th.TearDown()

	th.App.UpdateConfig(func(cfg *model.Config) {
		*cfg.ServiceSettings.AllowedUntrustedInternalConnections = "localhost,127.0.0.1"
	})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request model.PostActionIntegrationRequest
		jsonErr := json.NewDecoder(r.Body).Decode(&request)
		assert.NoError(t, jsonErr)

		fmt.Fprintf(w, `{
			"update": {
				"message": "updated",
				"has_reactions": true,
				"is_pinned": false,
				"props": {
					"from_webhook":"true",
					"override_username":"new_override_user",
					"override_icon_url":"new_override_icon",
					"A":"AA"
				}
			},
			"ephemeral_text": "foo"
		}`)
	}))
	defer ts.Close()

	interactivePost := model.Post{
		Message:       "Interactive post",
		ChannelId:     th.BasicChannel.Id,
		PendingPostId: model.NewId() + ":" + fmt.Sprint(model.GetMillis()),
		UserId:        th.BasicUser.Id,
		HasReactions:  false,
		IsPinned:      true,
		Props: model.StringInterface{
			model.PostPropsAttachments: []*model.SlackAttachment{
				{
					Text: "hello",
					Actions: []*model.PostAction{
						{
							Type:       model.PostActionTypeSelect,
							Name:       "action",
							DataSource: model.PostActionDataSourceUsers,
							Integration: &model.PostActionIntegration{
								Context: model.StringInterface{
									"s": "foo",
									"n": 3,
								},
								URL: ts.URL,
							},
						},
					},
				},
			},
			model.PostPropsOverrideIconURL: "old_override_icon",
			model.PostPropsFromWebhook:     "false",
			"B":                            "BB",
		},
	}

	post, err := th.App.CreatePostAsUser(th.Context, &interactivePost, "", true)
	require.Nil(t, err)
	attachments, ok := post.GetProp(model.PostPropsAttachments).([]*model.SlackAttachment)
	require.True(t, ok)

	clientTriggerId, err := th.App.DoPostActionWithCookie(th.Context, post.Id, attachments[0].Actions[0].Id, th.BasicUser.Id, "", nil)
	require.Nil(t, err)
	assert.Len(t, clientTriggerId, 26)

	newPost, nErr := th.App.Srv().Store().Post().GetSingle(th.Context, post.Id, false)
	require.NoError(t, nErr)

	assert.True(t, newPost.IsPinned)
	assert.False(t, newPost.HasReactions)
	assert.Nil(t, newPost.GetProp("B"))
	assert.Nil(t, newPost.GetProp(model.PostPropsOverrideUsername))
	assert.Equal(t, "AA", newPost.GetProp("A"))
	assert.Equal(t, "old_override_icon", newPost.GetProp(model.PostPropsOverrideIconURL))
	assert.Equal(t, "false", newPost.GetProp(model.PostPropsFromWebhook))
}

func TestSubmitInteractiveDialog(t *testing.T) {
	mainHelper.Parallel(t)
	th := Setup(t).InitBasic()
	defer th.TearDown()

	th.App.UpdateConfig(func(cfg *model.Config) {
		*cfg.ServiceSettings.AllowedUntrustedInternalConnections = "localhost,127.0.0.1"
	})

	submit := model.SubmitDialogRequest{
		UserId:     th.BasicUser.Id,
		ChannelId:  th.BasicChannel.Id,
		TeamId:     th.BasicTeam.Id,
		CallbackId: "someid",
		State:      "somestate",
		Submission: map[string]any{
			"name1": "value1",
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request model.SubmitDialogRequest
		err := json.NewDecoder(r.Body).Decode(&request)
		require.NoError(t, err)
		assert.NotNil(t, request)

		assert.Equal(t, request.URL, "")
		assert.Equal(t, request.UserId, submit.UserId)
		assert.Equal(t, request.ChannelId, submit.ChannelId)
		assert.Equal(t, request.TeamId, submit.TeamId)
		assert.Equal(t, request.CallbackId, submit.CallbackId)
		assert.Equal(t, request.State, submit.State)
		val, ok := request.Submission["name1"].(string)
		require.True(t, ok)
		assert.Equal(t, "value1", val)

		resp := model.SubmitDialogResponse{
			Error:  "some generic error",
			Errors: map[string]string{"name1": "some error"},
		}

		b, err := json.Marshal(resp)
		require.NoError(t, err)

		_, err = w.Write(b)
		require.NoError(t, err)
	}))
	defer ts.Close()

	setupPluginAPITest(t,
		`
		package main

		import (
			"net/http"
			"encoding/json"

			"github.com/mattermost/mattermost/server/public/plugin"
			"github.com/mattermost/mattermost/server/public/model"
		)

		type MyPlugin struct {
			plugin.MattermostPlugin
		}

		func (p *MyPlugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
			errReply := "some error"
 			if r.URL.Query().Get("abc") == "xyz" {
				errReply = "some other error"
			}
			response := &model.SubmitDialogResponse{
				Errors: map[string]string{"name1": errReply},
			}
			w.WriteHeader(http.StatusOK)
			responseJSON, _ := json.Marshal(response)
			_, _ = w.Write(responseJSON)
		}

		func main() {
			plugin.ClientMain(&MyPlugin{})
		}
		`, `{"id": "myplugin", "server": {"executable": "backend.exe"}}`, "myplugin", th.App, th.Context)

	hooks, err2 := th.App.GetPluginsEnvironment().HooksForPlugin("myplugin")
	require.NoError(t, err2)
	require.NotNil(t, hooks)

	submit.URL = ts.URL

	resp, err := th.App.SubmitInteractiveDialog(th.Context, submit)
	assert.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "some generic error", resp.Error)
	assert.Equal(t, "some error", resp.Errors["name1"])

	submit.URL = ""
	resp, err = th.App.SubmitInteractiveDialog(th.Context, submit)
	assert.NotNil(t, err)
	assert.Nil(t, resp)

	th.App.UpdateConfig(func(cfg *model.Config) {
		*cfg.ServiceSettings.AllowedUntrustedInternalConnections = ""
		*cfg.ServiceSettings.SiteURL = ts.URL
	})

	submit.URL = "/notvalid/myplugin/myaction"
	resp, err = th.App.SubmitInteractiveDialog(th.Context, submit)
	assert.NotNil(t, err)
	require.Nil(t, resp)

	submit.URL = "/plugins/myplugin/myaction"
	resp, err = th.App.SubmitInteractiveDialog(th.Context, submit)
	assert.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "some error", resp.Errors["name1"])

	submit.URL = "/plugins/myplugin/myaction?abc=xyz"
	resp, err = th.App.SubmitInteractiveDialog(th.Context, submit)
	assert.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "some other error", resp.Errors["name1"])
}

func TestPostActionRelativeURL(t *testing.T) {
	mainHelper.Parallel(t)
	th := Setup(t).InitBasic()
	defer th.TearDown()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request model.PostActionIntegrationRequest
		jsonErr := json.NewDecoder(r.Body).Decode(&request)
		assert.NoError(t, jsonErr)
		fmt.Fprintf(w, `{"post": {"message": "updated"}, "ephemeral_text": "foo"}`)
	}))
	defer ts.Close()

	t.Run("invalid relative URL", func(t *testing.T) {
		th.App.UpdateConfig(func(cfg *model.Config) {
			*cfg.ServiceSettings.AllowedUntrustedInternalConnections = ""
			*cfg.ServiceSettings.SiteURL = ts.URL
		})

		interactivePost := model.Post{
			Message:       "Interactive post",
			ChannelId:     th.BasicChannel.Id,
			PendingPostId: model.NewId() + ":" + fmt.Sprint(model.GetMillis()),
			UserId:        th.BasicUser.Id,
			Props: model.StringInterface{
				model.PostPropsAttachments: []*model.SlackAttachment{
					{
						Text: "hello",
						Actions: []*model.PostAction{
							{
								Type: model.PostActionTypeButton,
								Name: "action",
								Integration: &model.PostActionIntegration{
									URL: "/notaplugin/some/path",
								},
							},
						},
					},
				},
			},
		}

		post, err := th.App.CreatePostAsUser(th.Context, &interactivePost, "", true)
		require.Nil(t, err)
		attachments, ok := post.GetProp(model.PostPropsAttachments).([]*model.SlackAttachment)
		require.True(t, ok)
		require.NotEmpty(t, attachments[0].Actions)
		require.NotEmpty(t, attachments[0].Actions[0].Id)

		_, err = th.App.DoPostActionWithCookie(th.Context, post.Id, attachments[0].Actions[0].Id, th.BasicUser.Id, "", nil)
		require.NotNil(t, err)
	})

	t.Run("valid relative URL without SiteURL set", func(t *testing.T) {
		th.App.UpdateConfig(func(cfg *model.Config) {
			*cfg.ServiceSettings.AllowedUntrustedInternalConnections = ""
			*cfg.ServiceSettings.SiteURL = ""
		})

		interactivePost := model.Post{
			Message:       "Interactive post",
			ChannelId:     th.BasicChannel.Id,
			PendingPostId: model.NewId() + ":" + fmt.Sprint(model.GetMillis()),
			UserId:        th.BasicUser.Id,
			Props: model.StringInterface{
				model.PostPropsAttachments: []*model.SlackAttachment{
					{
						Text: "hello",
						Actions: []*model.PostAction{
							{
								Type: model.PostActionTypeButton,
								Name: "action",
								Integration: &model.PostActionIntegration{
									URL: "/plugins/myplugin/myaction",
								},
							},
						},
					},
				},
			},
		}

		post, err := th.App.CreatePostAsUser(th.Context, &interactivePost, "", true)
		require.Nil(t, err)
		attachments, ok := post.GetProp(model.PostPropsAttachments).([]*model.SlackAttachment)
		require.True(t, ok)
		require.NotEmpty(t, attachments[0].Actions)
		require.NotEmpty(t, attachments[0].Actions[0].Id)

		_, err = th.App.DoPostActionWithCookie(th.Context, post.Id, attachments[0].Actions[0].Id, th.BasicUser.Id, "", nil)
		require.NotNil(t, err)
	})

	t.Run("valid relative URL with SiteURL set", func(t *testing.T) {
		th.App.UpdateConfig(func(cfg *model.Config) {
			*cfg.ServiceSettings.AllowedUntrustedInternalConnections = ""
			*cfg.ServiceSettings.SiteURL = ts.URL
		})

		interactivePost := model.Post{
			Message:       "Interactive post",
			ChannelId:     th.BasicChannel.Id,
			PendingPostId: model.NewId() + ":" + fmt.Sprint(model.GetMillis()),
			UserId:        th.BasicUser.Id,
			Props: model.StringInterface{
				model.PostPropsAttachments: []*model.SlackAttachment{
					{
						Text: "hello",
						Actions: []*model.PostAction{
							{
								Type: model.PostActionTypeButton,
								Name: "action",
								Integration: &model.PostActionIntegration{
									URL: "/plugins/myplugin/myaction",
								},
							},
						},
					},
				},
			},
		}

		post, err := th.App.CreatePostAsUser(th.Context, &interactivePost, "", true)
		require.Nil(t, err)
		attachments, ok := post.GetProp(model.PostPropsAttachments).([]*model.SlackAttachment)
		require.True(t, ok)
		require.NotEmpty(t, attachments[0].Actions)
		require.NotEmpty(t, attachments[0].Actions[0].Id)

		_, err = th.App.DoPostActionWithCookie(th.Context, post.Id, attachments[0].Actions[0].Id, th.BasicUser.Id, "", nil)
		require.NotNil(t, err)
	})

	t.Run("valid (but dirty) relative URL with SiteURL set", func(t *testing.T) {
		th.App.UpdateConfig(func(cfg *model.Config) {
			*cfg.ServiceSettings.AllowedUntrustedInternalConnections = ""
			*cfg.ServiceSettings.SiteURL = ts.URL
		})

		interactivePost := model.Post{
			Message:       "Interactive post",
			ChannelId:     th.BasicChannel.Id,
			PendingPostId: model.NewId() + ":" + fmt.Sprint(model.GetMillis()),
			UserId:        th.BasicUser.Id,
			Props: model.StringInterface{
				model.PostPropsAttachments: []*model.SlackAttachment{
					{
						Text: "hello",
						Actions: []*model.PostAction{
							{
								Type: model.PostActionTypeButton,
								Name: "action",
								Integration: &model.PostActionIntegration{
									URL: "//plugins/myplugin///myaction",
								},
							},
						},
					},
				},
			},
		}

		post, err := th.App.CreatePostAsUser(th.Context, &interactivePost, "", true)
		require.Nil(t, err)
		attachments, ok := post.GetProp(model.PostPropsAttachments).([]*model.SlackAttachment)
		require.True(t, ok)
		require.NotEmpty(t, attachments[0].Actions)
		require.NotEmpty(t, attachments[0].Actions[0].Id)

		_, err = th.App.DoPostActionWithCookie(th.Context, post.Id, attachments[0].Actions[0].Id, th.BasicUser.Id, "", nil)
		require.NotNil(t, err)
	})

	t.Run("valid relative URL with SiteURL set and no leading slash", func(t *testing.T) {
		th.App.UpdateConfig(func(cfg *model.Config) {
			*cfg.ServiceSettings.AllowedUntrustedInternalConnections = ""
			*cfg.ServiceSettings.SiteURL = ts.URL
		})

		interactivePost := model.Post{
			Message:       "Interactive post",
			ChannelId:     th.BasicChannel.Id,
			PendingPostId: model.NewId() + ":" + fmt.Sprint(model.GetMillis()),
			UserId:        th.BasicUser.Id,
			Props: model.StringInterface{
				model.PostPropsAttachments: []*model.SlackAttachment{
					{
						Text: "hello",
						Actions: []*model.PostAction{
							{
								Type: model.PostActionTypeButton,
								Name: "action",
								Integration: &model.PostActionIntegration{
									URL: "plugins/myplugin/myaction",
								},
							},
						},
					},
				},
			},
		}

		post, err := th.App.CreatePostAsUser(th.Context, &interactivePost, "", true)
		require.Nil(t, err)
		attachments, ok := post.GetProp(model.PostPropsAttachments).([]*model.SlackAttachment)
		require.True(t, ok)
		require.NotEmpty(t, attachments[0].Actions)
		require.NotEmpty(t, attachments[0].Actions[0].Id)

		_, err = th.App.DoPostActionWithCookie(th.Context, post.Id, attachments[0].Actions[0].Id, th.BasicUser.Id, "", nil)
		require.NotNil(t, err)
	})
}

func TestPostActionRelativePluginURL(t *testing.T) {
	mainHelper.Parallel(t)
	th := Setup(t).InitBasic()
	defer th.TearDown()

	setupPluginAPITest(t,
		`
		package main

		import (
			"net/http"
			"encoding/json"

			"github.com/mattermost/mattermost/server/public/plugin"
			"github.com/mattermost/mattermost/server/public/model"
		)

		type MyPlugin struct {
			plugin.MattermostPlugin
		}

		func (p *MyPlugin) 	ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
			response := &model.PostActionIntegrationResponse{}
			w.WriteHeader(http.StatusOK)
			responseJSON, _ := json.Marshal(response)
			_, _ = w.Write(responseJSON)
		}

		func main() {
			plugin.ClientMain(&MyPlugin{})
		}
		`, `{"id": "myplugin", "server": {"executable": "backend.exe"}}`, "myplugin", th.App, th.Context)

	hooks, err2 := th.App.GetPluginsEnvironment().HooksForPlugin("myplugin")
	require.NoError(t, err2)
	require.NotNil(t, hooks)

	t.Run("invalid relative URL", func(t *testing.T) {
		th.App.UpdateConfig(func(cfg *model.Config) {
			*cfg.ServiceSettings.AllowedUntrustedInternalConnections = ""
			*cfg.ServiceSettings.SiteURL = ""
		})

		interactivePost := model.Post{
			Message:       "Interactive post",
			ChannelId:     th.BasicChannel.Id,
			PendingPostId: model.NewId() + ":" + fmt.Sprint(model.GetMillis()),
			UserId:        th.BasicUser.Id,
			Props: model.StringInterface{
				model.PostPropsAttachments: []*model.SlackAttachment{
					{
						Text: "hello",
						Actions: []*model.PostAction{
							{
								Type: model.PostActionTypeButton,
								Name: "action",
								Integration: &model.PostActionIntegration{
									URL: "/notaplugin/some/path",
								},
							},
						},
					},
				},
			},
		}

		post, err := th.App.CreatePostAsUser(th.Context, &interactivePost, "", true)
		require.Nil(t, err)
		attachments, ok := post.GetProp(model.PostPropsAttachments).([]*model.SlackAttachment)
		require.True(t, ok)
		require.NotEmpty(t, attachments[0].Actions)
		require.NotEmpty(t, attachments[0].Actions[0].Id)

		_, err = th.App.DoPostActionWithCookie(th.Context, post.Id, attachments[0].Actions[0].Id, th.BasicUser.Id, "", nil)
		require.NotNil(t, err)
	})

	t.Run("valid relative URL", func(t *testing.T) {
		th.App.UpdateConfig(func(cfg *model.Config) {
			*cfg.ServiceSettings.AllowedUntrustedInternalConnections = ""
			*cfg.ServiceSettings.SiteURL = ""
		})

		interactivePost := model.Post{
			Message:       "Interactive post",
			ChannelId:     th.BasicChannel.Id,
			PendingPostId: model.NewId() + ":" + fmt.Sprint(model.GetMillis()),
			UserId:        th.BasicUser.Id,
			Props: model.StringInterface{
				model.PostPropsAttachments: []*model.SlackAttachment{
					{
						Text: "hello",
						Actions: []*model.PostAction{
							{
								Type: model.PostActionTypeButton,
								Name: "action",
								Integration: &model.PostActionIntegration{
									URL: "/plugins/myplugin/myaction",
								},
							},
						},
					},
				},
			},
		}

		post, err := th.App.CreatePostAsUser(th.Context, &interactivePost, "", true)
		require.Nil(t, err)
		attachments, ok := post.GetProp(model.PostPropsAttachments).([]*model.SlackAttachment)
		require.True(t, ok)
		require.NotEmpty(t, attachments[0].Actions)
		require.NotEmpty(t, attachments[0].Actions[0].Id)

		_, err = th.App.DoPostActionWithCookie(th.Context, post.Id, attachments[0].Actions[0].Id, th.BasicUser.Id, "", nil)
		require.Nil(t, err)
	})

	t.Run("valid (but dirty) relative URL", func(t *testing.T) {
		th.App.UpdateConfig(func(cfg *model.Config) {
			*cfg.ServiceSettings.AllowedUntrustedInternalConnections = ""
			*cfg.ServiceSettings.SiteURL = ""
		})

		interactivePost := model.Post{
			Message:       "Interactive post",
			ChannelId:     th.BasicChannel.Id,
			PendingPostId: model.NewId() + ":" + fmt.Sprint(model.GetMillis()),
			UserId:        th.BasicUser.Id,
			Props: model.StringInterface{
				model.PostPropsAttachments: []*model.SlackAttachment{
					{
						Text: "hello",
						Actions: []*model.PostAction{
							{
								Type: model.PostActionTypeButton,
								Name: "action",
								Integration: &model.PostActionIntegration{
									URL: "//plugins/myplugin///myaction",
								},
							},
						},
					},
				},
			},
		}

		post, err := th.App.CreatePostAsUser(th.Context, &interactivePost, "", true)
		require.Nil(t, err)
		attachments, ok := post.GetProp(model.PostPropsAttachments).([]*model.SlackAttachment)
		require.True(t, ok)
		require.NotEmpty(t, attachments[0].Actions)
		require.NotEmpty(t, attachments[0].Actions[0].Id)

		_, err = th.App.DoPostActionWithCookie(th.Context, post.Id, attachments[0].Actions[0].Id, th.BasicUser.Id, "", nil)
		require.Nil(t, err)
	})

	t.Run("valid relative URL and no leading slash", func(t *testing.T) {
		th.App.UpdateConfig(func(cfg *model.Config) {
			*cfg.ServiceSettings.AllowedUntrustedInternalConnections = ""
			*cfg.ServiceSettings.SiteURL = ""
		})

		interactivePost := model.Post{
			Message:       "Interactive post",
			ChannelId:     th.BasicChannel.Id,
			PendingPostId: model.NewId() + ":" + fmt.Sprint(model.GetMillis()),
			UserId:        th.BasicUser.Id,
			Props: model.StringInterface{
				model.PostPropsAttachments: []*model.SlackAttachment{
					{
						Text: "hello",
						Actions: []*model.PostAction{
							{
								Type: model.PostActionTypeButton,
								Name: "action",
								Integration: &model.PostActionIntegration{
									URL: "plugins/myplugin/myaction",
								},
							},
						},
					},
				},
			},
		}

		post, err := th.App.CreatePostAsUser(th.Context, &interactivePost, "", true)
		require.Nil(t, err)
		attachments, ok := post.GetProp(model.PostPropsAttachments).([]*model.SlackAttachment)
		require.True(t, ok)
		require.NotEmpty(t, attachments[0].Actions)
		require.NotEmpty(t, attachments[0].Actions[0].Id)

		_, err = th.App.DoPostActionWithCookie(th.Context, post.Id, attachments[0].Actions[0].Id, th.BasicUser.Id, "", nil)
		require.Nil(t, err)
	})
}

func TestDoPluginRequest(t *testing.T) {
	mainHelper.Parallel(t)
	th := Setup(t)
	defer th.TearDown()

	th.App.UpdateConfig(func(cfg *model.Config) {
		*cfg.ServiceSettings.AllowedUntrustedInternalConnections = "localhost,127.0.0.1"
	})

	setupPluginAPITest(t,
		`
		package main

		import (
			"net/http"
			"reflect"
			"sort"

			"github.com/mattermost/mattermost/server/public/plugin"
		)

		type MyPlugin struct {
			plugin.MattermostPlugin
		}

		func (p *MyPlugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()
			if q.Get("abc") != "xyz" {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("could not find param abc=xyz"))
				return
			}

			multiple := q["multiple"]
			if len(multiple) != 3 {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("param multiple should have 3 values"))
				return
			}
			sort.Strings(multiple)
			if !reflect.DeepEqual(multiple, []string{"1 first", "2 second", "3 third"}) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("param multiple not correct"))
				return
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		}

		func main() {
			plugin.ClientMain(&MyPlugin{})
		}
		`, `{"id": "myplugin", "server": {"executable": "backend.exe"}}`, "myplugin", th.App, th.Context)

	hooks, err2 := th.App.GetPluginsEnvironment().HooksForPlugin("myplugin")
	require.NoError(t, err2)
	require.NotNil(t, hooks)

	resp, err := th.App.doPluginRequest(th.Context, "GET", "/plugins/myplugin", nil, nil)
	assert.Nil(t, err)
	require.NotNil(t, resp)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "could not find param abc=xyz", string(body))

	resp, err = th.App.doPluginRequest(th.Context, "GET", "/plugins/myplugin?abc=xyz", nil, nil)
	assert.Nil(t, err)
	require.NotNil(t, resp)
	body, _ = io.ReadAll(resp.Body)
	assert.Equal(t, "param multiple should have 3 values", string(body))

	resp, err = th.App.doPluginRequest(th.Context, "GET", "/plugins/myplugin",
		url.Values{"abc": []string{"xyz"}, "multiple": []string{"1 first", "2 second", "3 third"}}, nil)
	assert.Nil(t, err)
	require.NotNil(t, resp)
	body, _ = io.ReadAll(resp.Body)
	assert.Equal(t, "OK", string(body))

	resp, err = th.App.doPluginRequest(th.Context, "GET", "/plugins/myplugin?abc=xyz&multiple=1%20first",
		url.Values{"multiple": []string{"2 second", "3 third"}}, nil)
	assert.Nil(t, err)
	require.NotNil(t, resp)
	body, _ = io.ReadAll(resp.Body)
	assert.Equal(t, "OK", string(body))

	resp, err = th.App.doPluginRequest(th.Context, "GET", "/plugins/myplugin?abc=xyz&multiple=1%20first&multiple=3%20third",
		url.Values{"multiple": []string{"2 second"}}, nil)
	assert.Nil(t, err)
	require.NotNil(t, resp)
	body, _ = io.ReadAll(resp.Body)
	assert.Equal(t, "OK", string(body))

	resp, err = th.App.doPluginRequest(th.Context, "GET", "/plugins/myplugin?multiple=1%20first&multiple=3%20third",
		url.Values{"multiple": []string{"2 second"}, "abc": []string{"xyz"}}, nil)
	assert.Nil(t, err)
	require.NotNil(t, resp)
	body, _ = io.ReadAll(resp.Body)
	assert.Equal(t, "OK", string(body))

	resp, err = th.App.doPluginRequest(th.Context, "GET", "/plugins/myplugin?multiple=1%20first&multiple=3%20third",
		url.Values{"multiple": []string{"4 fourth"}, "abc": []string{"xyz"}}, nil)
	assert.Nil(t, err)
	require.NotNil(t, resp)
	body, _ = io.ReadAll(resp.Body)
	assert.Equal(t, "param multiple not correct", string(body))
}
