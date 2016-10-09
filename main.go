// DP-Download project main.go

package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/kardianos/osext"

	"github.com/bwmarrin/discordgo"
)

var globalMessageLimit = 100
var errJSONUnmarshal = errors.New("json unmarshal")

type counter struct {
	messageCounter    int
	attachmentCounter int
	embedCounter      int
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Fatal(r)
		}
	}()
	// Use discordgo.New(Token) to just use a token for login.
	dg, err := login()
	if err != nil {
		fmt.Println("Error creating DiscordGo session, ", err)
		return
	}

	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening connection, ", err)
		return
	}

	fmt.Println("Bot is running. Press CTRL-C to exit.")

	// New stuff. What do you want to download?
	// Private or guild?
	// 	-Private:
	//		-Get list of private channels.
	//		-One, multiple, or all?
	//			-One
	//				-ID or username+number
	//				-Check through each in list for ID or username+number
	//			-All
	//				-Get all.
	//			-Multiple
	//				-Do this later...
	// 	-Guild:
	//		-What is the channelID?
	channelIds := getChannelIDs(dg)
	baseFilePath := getBasePath(dg)
	if len(channelIds) > 0 {
		for i := 0; i < len(channelIds); i++ {
			getAllMessages(dg, channelIds[i], baseFilePath)

		}
	} else {
		fmt.Println("Did not find any channel IDs. Exiting.")
	}
	//
	fmt.Println("Done with download procedures.")
	fmt.Print("Press Enter to close.")
	reader := bufio.NewReader(os.Stdin)
	_, err = reader.ReadString('\n')
	return
}

func getChannelIDs(s *discordgo.Session) []string {
	var channelIds []string
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Do you want private channels?: (Y/n) ")
	tempResponse, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	tempResponse = strings.ToLower(strings.TrimSpace(tempResponse))
	if tempResponse == "n" {
		// If it is n.
		fmt.Print("Channel ID: ")
		tempResponse, err = reader.ReadString('\n')
		if err != nil {
			fmt.Print("Unable to read channel id.")
			panic(err)
		}
		channelIds = append(channelIds, strings.TrimSpace(tempResponse))

	} else if tempResponse == "y" || tempResponse == "" {
		// If it is either y or empty.
		fmt.Println("Getting private channels associated with account.")
		privateList, err := s.UserChannels()
		if err != nil {
			fmt.Println("Failed to fetch private channels.")
			panic(err)
		}
		fmt.Println("Private channels retrieved.")

		fmt.Print(
			"Do you want to download all of your private channels?: (N/y) ")
		tempResponse, err := reader.ReadString('\n')
		if err != nil {
			panic(err)
		}
		tempResponse = strings.ToLower(strings.TrimSpace(tempResponse))
		if tempResponse == "n" || tempResponse == "" {
			usrChnnls, err := s.UserChannels()
			if len(usrChnnls) > 0 {
				for chnlIterator := 0; chnlIterator < len(usrChnnls); chnlIterator++ {
					if usrChnnls[chnlIterator].IsPrivate {
						fmt.Printf("%v %v %v\n", usrChnnls[chnlIterator].Recipient.Username, usrChnnls[chnlIterator].Recipient.Discriminator, usrChnnls[chnlIterator].ID)
					}
				}
			}
			// If they don't want to download them all.
			fmt.Print("Do you have the ID?: (Y/n) ")
			tempResponse, err = reader.ReadString('\n')
			if err != nil {
				panic(err)
			}
			tempResponse = strings.ToLower(strings.TrimSpace(tempResponse))
			if tempResponse == "y" || tempResponse == "" {
				// If they have an id already.

				fmt.Print("Channel ID: ")
				tempResponse, err = reader.ReadString('\n')
				if err != nil {
					fmt.Print("Unable to read channel id.")
					panic(err)
				}
				channelIds = append(channelIds, strings.TrimSpace(tempResponse))
			} else if tempResponse == "n" {
				// If they don't have it.

				// Should be last one to finish!
				fmt.Print("Username: ")
				tempUsername, err := reader.ReadString('\n')
				if err != nil {
					fmt.Print("Unable to read username.")
					panic(err)
				}
				tempUsername = strings.TrimSpace(tempUsername)
				fmt.Print("Discriminator: ")
				tempDiscriminator, err := reader.ReadString('\n')
				if err != nil {
					panic(err)
				}
				tempDiscriminator = strings.TrimSpace(tempDiscriminator)
				// If it is neither y or n.
				fmt.Print("Unable to read discriminator.")

				for i := 0; i < len(privateList); i++ {
					if privateList[i].Recipient.Username == tempUsername &&
						privateList[i].Recipient.Discriminator ==
							tempDiscriminator {
						channelIds = append(channelIds,
							privateList[i].ID)
					}
				}
			}

		} else if tempResponse == "y" {
			// If they want them all
			for i := 0; i < len(privateList); i++ {
				channelIds = append(channelIds, privateList[i].ID)
			}
		} else {
			// If it is neither y or n.
			fmt.Println("Response not found.")
		}
	} else {
		// If it is neither Y or n.
		fmt.Println("Response not found.")
	}
	return channelIds
}

func login() (*discordgo.Session, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter email address: ")
	tempString, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	email := strings.TrimSpace(tempString)
	fmt.Print("Enter password: ")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		panic(err)
	}
	fmt.Println("")
	password := strings.TrimSpace(string(bytePassword))
	var token string
	dg, err := discordgo.New(email, password, token)

	return dg, err

}

func getBasePath(dg *discordgo.Session) string {
	baseFilePath, err := osext.ExecutableFolder()
	if err != nil {
		panic(err)
	}
	baseFilePath = filepath.Join(baseFilePath, "files")
	userData, err := dg.User("@me")
	if err != nil {
		panic("Unable to get user info.")
	}

	baseFilePath = filepath.Join(baseFilePath, userData.Username+userData.Discriminator)
	err = os.MkdirAll(baseFilePath, os.ModePerm)
	if err != nil {
		fmt.Println("Error 1: Unable to make path.")
		panic(err)
	}

	return baseFilePath

}

func getAllMessages(s *discordgo.Session, chatID string, baseFilePath string) {

	chnnl, err := s.Channel(chatID)

	if err != nil {
		panic(err)
	}

	fmt.Println("Starting download process.")
	switch chnnl.IsPrivate {
	case true:
		// It is private channel. Print recipient.
		fmt.Printf("Fetching channel: %v\n", chnnl.Recipient.Username)
	case false:
		fmt.Printf("Fetching channel: %v\n", chnnl.Name)
	default:
		panic("Channel has neither a name nor a recipient.")
	}

	lastMSG := chnnl.LastMessageID

	messageCounter := counter{0, 0, 0}
	messageString := ""
	attachmentString := ""
	embedString := ""

	if chnnl.IsPrivate == true {
		baseFilePath = filepath.Join(baseFilePath, "private")
		baseFilePath = filepath.Join(baseFilePath, chnnl.Recipient.Username)
		err = os.MkdirAll(baseFilePath, os.ModePerm)
		if err != nil {
			fmt.Println("Error 1")
			return
		}
	} else {
		baseFilePath = filepath.Join(baseFilePath, "guilds")
		baseFilePath = filepath.Join(baseFilePath, chnnl.GuildID)
		baseFilePath = filepath.Join(baseFilePath, chnnl.Name)
		err = os.MkdirAll(baseFilePath, os.ModePerm)
		if err != nil {
			fmt.Println("Error 1")
			return
		}
	}

	attachmentPath := filepath.Join(baseFilePath, "Attachments")
	err = os.MkdirAll(attachmentPath, os.ModePerm)
	if err != nil {
		fmt.Println("Error 2")
		return
	}
	embedPath := filepath.Join(baseFilePath, "Embeds")
	err = os.MkdirAll(embedPath, os.ModePerm)
	if err != nil {
		fmt.Println("Error 3")
		return
	}

	baseMessagesPath := filepath.Join(baseFilePath, "messages.csv")
	messagesFile, err := os.Create(baseMessagesPath)
	if err != nil {
		panic(err)
	}
	messageWriter := bufio.NewWriter(messagesFile)
	defer messageWriter.Flush()
	defer messagesFile.Close()

	baseAttachmentsPath := filepath.Join(baseFilePath, "attachments.csv")
	attachmentsFile, err := os.Create(baseAttachmentsPath)

	if err != nil {
		panic(err)
	}
	attachmentWriter := bufio.NewWriter(attachmentsFile)
	defer attachmentWriter.Flush()
	defer attachmentsFile.Close()

	baseEmbedPath := filepath.Join(baseFilePath, "embed.csv")
	embedsFile, err := os.Create(baseEmbedPath)
	if err != nil {
		panic(err)
	}
	embedWriter := bufio.NewWriter(embedsFile)
	defer embedWriter.Flush()
	defer embedsFile.Close()

	// Get the first message maybe? Now two!
	lastMSGListFull, err := getFirstMessage(chatID, 2, "", "", lastMSG, s)
	if err != nil {
		panic(err)
	}

	if len(lastMSGListFull) > 0 {
		for i := 0; i < len(lastMSGListFull); i++ {
			messageCounter = processOneMessage(i,
				messageCounter,
				messageString,
				lastMSGListFull,
				attachmentString,
				attachmentWriter,
				attachmentPath,
				embedString,
				embedWriter,
				embedPath,
				messageWriter,
			)
		}
	}

	if len(lastMSGListFull) > 1 {
		lastMSG = lastMSGListFull[1].ID

		// Start getting previous messages.
		msgList, err := s.ChannelMessages(chatID,
			globalMessageLimit,
			lastMSG,
			"")
		if err != nil {
			panic(err)
		}

		for len(msgList) == globalMessageLimit {
			// Do stuff with messages.
			//get each message from the array
			//replace mentions with names
			//concactenate mention roles
			//
			//
			// Open messages.csv, attachments.csv, and embed.csv files.
			// Make variables for the Attachments and Embed folders.
			for i := 0; i < len(msgList); i++ {
				messageCounter = processOneMessage(i,
					messageCounter,
					messageString,
					msgList,
					attachmentString,
					attachmentWriter,
					attachmentPath,
					embedString,
					embedWriter,
					embedPath,
					messageWriter,
				)
			}

			lastMSG = msgList[len(msgList)-1].ID
			time.Sleep(time.Millisecond * 1300)
			msgList, err = s.ChannelMessages(chatID,
				globalMessageLimit,
				lastMSG,
				"")

			if err != nil {
				panic(err)
			}
		}
		// Do stuff with last set.
		if len(msgList) > 0 {
			for i := 0; i < len(msgList); i++ {
				messageCounter = processOneMessage(i,
					messageCounter,
					messageString,
					msgList,
					attachmentString,
					attachmentWriter,
					attachmentPath,
					embedString,
					embedWriter,
					embedPath,
					messageWriter,
				)
			}
		}
		//
	}
	fmt.Printf("Fetched %v messages, %v attachments, and %v embedings.\n", messageCounter.messageCounter, messageCounter.attachmentCounter, messageCounter.embedCounter)
}

func processOneMessage(i int,
	messageCounter counter,
	messageString string,
	msgList []*discordgo.Message,
	attachmentString string,
	attachmentWriter *bufio.Writer,
	attachmentPath string,
	embedString string,
	embedWriter *bufio.Writer,
	embedPath string,
	messageWriter *bufio.Writer,

) counter {

	concatString := ""
	messageCounter.messageCounter++
	messageString = msgList[i].ID + "<" +
		msgList[i].ChannelID + "<" +
		html.EscapeString(
			strings.Replace(
				msgList[i].ContentWithMentionsReplaced(),
				"\n", "<br>", -1)) + "<" +
		msgList[i].Timestamp + "<" +
		msgList[i].EditedTimestamp + "<"
	if len(msgList[i].MentionRoles) > 0 {
		concatString = msgList[i].MentionRoles[0]
		if len(msgList[i].MentionRoles) > 1 {
			for j := 1; j < len(msgList[i].MentionRoles); j++ {
				concatString += " " + msgList[i].MentionRoles[j]
			}
		}
	}

	messageString += concatString + "<"
	messageString += fmt.Sprintf("%v", msgList[i].Tts) + "<" +
		fmt.Sprintf("%v", msgList[i].MentionEveryone) + "<" +
		msgList[i].Author.ID + "<" +
		msgList[i].Author.Email + "<" +
		msgList[i].Author.Username + "<" +
		msgList[i].Author.Avatar + "<" +
		msgList[i].Author.Discriminator + "<" +
		msgList[i].Author.Token + "<" +
		fmt.Sprintf("%v", msgList[i].Author.Verified) + "<" +
		fmt.Sprintf("%v", msgList[i].Author.MFAEnabled) + "<" +
		fmt.Sprintf("%v", msgList[i].Author.Bot) + "<"

	concatString = ""
	if len(msgList[i].Attachments) > 0 {
		concatString = msgList[i].Attachments[0].ID
		attachmentString = msgList[i].ID + "<" +
			msgList[i].Attachments[0].ID + "<" +
			msgList[i].Attachments[0].URL + "<" +
			msgList[i].Attachments[0].ProxyURL + "<" +
			msgList[i].Attachments[0].Filename + "<" +
			fmt.Sprintf("%v", msgList[i].Attachments[0].Width) + "<" +
			fmt.Sprintf("%v", msgList[i].Attachments[0].Height) + "<" +
			fmt.Sprintf("%v", msgList[i].Attachments[0].Size) + "<" +
			strconv.Itoa(messageCounter.attachmentCounter)

		fmt.Fprintln(attachmentWriter, attachmentString)
		attachmentWriter.Flush()
		// Save the attachment.
		//--------------------------------------------------------------
		attachmentData, err := http.Get(msgList[i].Attachments[0].URL)
		if err != nil {
			log.Fatal(err)
		}

		attachmentSavePath := filepath.Join(
			attachmentPath,
			strconv.Itoa(messageCounter.attachmentCounter)+path.Ext(msgList[i].Attachments[0].Filename))
		attachmentFile, err := os.Create(attachmentSavePath)
		defer attachmentFile.Close()
		if err != nil {
			log.Fatal(err)
		}

		_, err = io.Copy(attachmentFile, attachmentData.Body)
		if err != nil {
			log.Fatal(err)
		}
		//--------------------------------------------------------------
		messageCounter.attachmentCounter++

		if len(msgList[i].Attachments) > 1 {
			for j := 1; j < len(msgList[i].Attachments); j++ {
				messageCounter.attachmentCounter++
				concatString += " " + msgList[i].Attachments[j].ID
				attachmentString = msgList[i].ID + "<" +
					msgList[i].Attachments[j].ID + "<" +
					msgList[i].Attachments[j].URL + "<" +
					msgList[i].Attachments[j].ProxyURL + "<" +
					msgList[i].Attachments[j].Filename + "<" +
					fmt.Sprintf("%v", msgList[i].Attachments[j].Width) + "<" +
					fmt.Sprintf("%v", msgList[i].Attachments[j].Height) + "<" +
					fmt.Sprintf("%v", msgList[i].Attachments[j].Size) + "<" +
					strconv.Itoa(messageCounter.attachmentCounter)

				fmt.Fprintln(attachmentWriter, attachmentString)
				attachmentWriter.Flush()
				// Save the attachment.
				//--------------------------------------------------------------
				attachmentData, err := http.Get(msgList[i].Attachments[j].URL)
				if err != nil {
					log.Fatal(err)
				}

				attachmentSavePath := filepath.Join(
					attachmentPath,
					strconv.Itoa(messageCounter.attachmentCounter)+path.Ext(msgList[i].Attachments[j].Filename))
				attachmentFile, err := os.Create(attachmentSavePath)
				defer attachmentFile.Close()
				if err != nil {
					log.Fatal(err)
				}

				_, err = io.Copy(attachmentFile, attachmentData.Body)
				if err != nil {
					log.Fatal(err)
				}
				//--------------------------------------------------------------

				messageCounter.attachmentCounter++
			}
		}
	}
	messageString += concatString + "<"

	if len(msgList[i].Embeds) > 0 {
		concatString = msgList[i].Embeds[0].URL
		embedString = msgList[i].ID + "<" +
			msgList[i].Embeds[0].URL + "<" +
			msgList[i].Embeds[0].Type + "<" +
			msgList[i].Embeds[0].Title + "<" +
			msgList[i].Embeds[0].Description + "<"
		if msgList[i].Embeds[0].Thumbnail != nil {
			embedString += msgList[i].Embeds[0].Thumbnail.URL + "<" +
				msgList[i].Embeds[0].Thumbnail.ProxyURL + "<" +
				fmt.Sprintf("%v", msgList[i].Embeds[0].Thumbnail.Width) + "<" +
				fmt.Sprintf("%v", msgList[i].Embeds[0].Thumbnail.Height) + "<"
		} else {
			embedString += "<<<<"
		}
		if msgList[i].Embeds[0].Provider != nil {
			embedString += msgList[i].Embeds[0].Provider.URL + "<" +
				msgList[i].Embeds[0].Provider.Name + "<"

		} else {
			embedString += "<<"
		}
		if msgList[i].Embeds[0].Author != nil {
			embedString += msgList[i].Embeds[0].Author.URL + "<" +
				msgList[i].Embeds[0].Author.Name + "<"
		} else {
			embedString += "<<"
		}
		if msgList[i].Embeds[0].Video != nil {
			embedString += msgList[i].Embeds[0].Video.URL + "<" +
				fmt.Sprintf("%v", msgList[i].Embeds[0].Video.Width) + "<" +
				fmt.Sprintf("%v", msgList[i].Embeds[0].Video.Height) + "<"
		} else {
			embedString += "<<<"
		}
		embedString += "<" + strconv.Itoa(messageCounter.embedCounter)

		fmt.Fprintln(embedWriter, embedString)
		embedWriter.Flush()
		// Save the thumbnail.
		//--------------------------------------------------------------
		if msgList[i].Embeds[0].Thumbnail != nil {
			embedData, err := http.Get(msgList[i].Embeds[0].Thumbnail.URL)
			if err != nil {
				log.Fatal(err)
			}

			embedSavePath := filepath.Join(embedPath, strconv.Itoa(messageCounter.embedCounter)+".png")
			embedFile, err := os.Create(embedSavePath)
			defer embedFile.Close()
			if err != nil {
				log.Fatal(err)
			}

			_, err = io.Copy(embedFile, embedData.Body)
			if err != nil {
				log.Fatal(err)
			}
		}

		messageCounter.embedCounter++
		//--------------------------------------------------------------

		if len(msgList[i].Embeds) > 1 {
			for j := 1; j < len(msgList[i].Embeds); j++ {
				concatString = msgList[i].Embeds[0].URL
				embedString = msgList[i].ID + "<" +
					msgList[i].Embeds[j].URL + "<" +
					msgList[i].Embeds[j].Type + "<" +
					msgList[i].Embeds[j].Title + "<" +
					msgList[i].Embeds[j].Description + "<"
				if msgList[i].Embeds[j].Thumbnail != nil {
					embedString += msgList[i].Embeds[j].Thumbnail.URL + "<" +
						msgList[i].Embeds[j].Thumbnail.ProxyURL + "<" +
						fmt.Sprintf("%v",
							msgList[i].Embeds[j].Thumbnail.Width) + "<" +
						fmt.Sprintf("%v",
							msgList[i].Embeds[j].Thumbnail.Height) + "<"
				} else {
					embedString += "<<<<"
				}
				if msgList[i].Embeds[j].Provider != nil {
					embedString += msgList[i].Embeds[j].Provider.URL + "<" +
						msgList[i].Embeds[j].Provider.Name + "<"

				} else {
					embedString += "<<"
				}
				if msgList[i].Embeds[j].Author != nil {
					embedString += msgList[i].Embeds[j].Author.URL + "<" +
						msgList[i].Embeds[j].Author.Name + "<"
				} else {
					embedString += "<<"
				}
				if msgList[i].Embeds[j].Video != nil {
					embedString += msgList[i].Embeds[j].Video.URL + "<" +
						fmt.Sprintf("%v",
							msgList[i].Embeds[j].Video.Width) + "<" +
						fmt.Sprintf("%v",
							msgList[i].Embeds[j].Video.Height)
				} else {
					embedString += "<<<"
				}
				embedString += "<" + strconv.Itoa(messageCounter.embedCounter)

				fmt.Fprintln(embedWriter, embedString)
				embedWriter.Flush()
				// Save the thumbnail.
				//--------------------------------------------------------------
				if msgList[i].Embeds[j].Thumbnail != nil {
					embedData, err :=
						http.Get(msgList[i].Embeds[j].Thumbnail.URL)
					if err != nil {
						log.Fatal(err)
					}

					embedSavePath :=
						filepath.Join(embedPath,
							strconv.Itoa(messageCounter.embedCounter)+".png")
					embedFile, err := os.Create(embedSavePath)
					defer embedFile.Close()
					if err != nil {
						log.Fatal(err)
					}

					_, err = io.Copy(embedFile, embedData.Body)
					if err != nil {
						log.Fatal(err)
					}
				}
				//--------------------------------------------------------------

				messageCounter.embedCounter++
			}
		}

	}
	messageString += concatString
	fmt.Fprintln(messageWriter, messageString)
	messageWriter.Flush()

	return messageCounter
}

func unmarshal(data []byte, v interface{}) error {
	err := json.Unmarshal(data, v)
	if err != nil {
		return errJSONUnmarshal
	}

	return nil
}

func getFirstMessage(channelID string,
	limit int, beforeID, afterID string,
	aroundID string,
	s *discordgo.Session) (st []*discordgo.Message, err error) {

	uri := discordgo.EndpointChannelMessages(channelID)

	v := url.Values{}
	if limit > 0 {
		v.Set("limit", strconv.Itoa(limit))
	}
	if afterID != "" {
		v.Set("after", afterID)
	}
	if beforeID != "" {
		v.Set("before", beforeID)
	}
	if aroundID != "" {
		v.Set("around", aroundID)
	}
	if len(v) > 0 {
		uri = fmt.Sprintf("%s?%s", uri, v.Encode())
	}

	body, err := s.Request("GET", uri, nil)
	if err != nil {
		return
	}

	err = unmarshal(body, &st)
	return
}
