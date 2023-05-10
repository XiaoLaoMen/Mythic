package grpc

import (
	"errors"
	"fmt"
	"github.com/its-a-feature/Mythic/grpc/services"
	"github.com/its-a-feature/Mythic/logging"
	"io"
)

func (t *translationContainerServer) GenerateEncryptionKeys(stream services.TranslationContainer_GenerateEncryptionKeysServer) error {
	clientName := ""
	// initially wait for a request from the other side with blank data to indicate who is talking to us
	if initial, err := stream.Recv(); err == io.EOF {
		logging.LogDebug("Client closed before ever sending anything, err is EOF")
		return nil // the client closed before ever sending anything
	} else if err != nil {
		logging.LogError(err, "Client ran into an error before sending anything")
		return err
	} else {
		clientName = initial.GetTranslationContainerName()
		if getMessageToSend, sendBackMessageResponse, err := t.addNewGenerateKeysClient(clientName); err != nil {
			logging.LogError(err, "Failed to add new channels to listen for connection")
			return err
		} else {
			logging.LogDebug("Got translation container name from remote connection", "name", clientName)
			for {
				select {
				case <-stream.Context().Done():
					logging.LogError(nil, fmt.Sprintf("client disconnected: %s", clientName))
					t.SetGenerateKeysChannelExited(clientName)
					return errors.New(fmt.Sprintf("client disconnected: %s", clientName))
				case msgToSend, ok := <-getMessageToSend:
					if !ok {
						logging.LogError(nil, "got !ok from messageToSend, channel was closed")
						t.SetGenerateKeysChannelExited(clientName)
						return nil
					} else {
						if err = stream.Send(&msgToSend); err != nil {
							logging.LogError(err, "Failed to send message through stream to translation container")
							sendBackMessageResponse <- services.TrGenerateEncryptionKeysMessageResponse{
								Success:                  false,
								Error:                    err.Error(),
								TranslationContainerName: clientName,
							}
							t.SetGenerateKeysChannelExited(clientName)
							return err
						} else if resp, err := stream.Recv(); err == io.EOF {
							// cleanup the connection channels first before returning
							sendBackMessageResponse <- services.TrGenerateEncryptionKeysMessageResponse{
								Success:                  false,
								Error:                    err.Error(),
								TranslationContainerName: clientName,
							}
							t.SetGenerateKeysChannelExited(clientName)
							return nil
						} else if err != nil {
							// cleanup the connection channels first before returning
							logging.LogError(err, "Failed to read from translation container")
							sendBackMessageResponse <- services.TrGenerateEncryptionKeysMessageResponse{
								Success:                  false,
								Error:                    err.Error(),
								TranslationContainerName: clientName,
							}
							t.SetGenerateKeysChannelExited(clientName)
							return err
						} else {
							sendBackMessageResponse <- *resp
						}
					}
				}
			}
		}
	}
}
