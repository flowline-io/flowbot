package types

import (
	"fmt"
	"testing"
)

func TestMsgBuilder(t *testing.T) {
	builder := MsgBuilder{}
	builder.AppendText("Hi,this is bold\n", TextOption{IsBold: true})
	builder.AppendText("Hi,this is italic\n", TextOption{IsItalic: true})
	builder.AppendText("Hi,this is deleted\n", TextOption{IsDeleted: true})

	builder.AppendText("int a=100;\nint b=a*100-90;\n", TextOption{IsCode: true})
	builder.AppendText("https://google.com\n", TextOption{IsLink: true})
	builder.AppendText("demo.com\n", TextOption{IsLink: true})
	builder.AppendText("@user\n", TextOption{IsMention: true})
	builder.AppendText("#tag\n", TextOption{IsHashTag: true})
	builder.AppendText("\n\nnext is image\n", TextOption{})
	builder.AppendImage("a.png", ImageOption{Mime: "image/png"})
	builder.AppendText("\n\nnext is file\n", TextOption{})
	builder.AppendFile("a.txt", FileOption{Mime: "text/plain"})
	builder.AppendAttachment("a.zip", AttachmentOption{Mime: "application/zip"})

	builder.AppendText("What's your gender?", TextOption{IsBold: true, IsForm: true})
	builder.AppendText("Male", TextOption{IsButton: true, ButtonDataName: "male", ButtonDataVal: "male", ButtonDataAct: "pub"})
	builder.AppendText("Female", TextOption{IsButton: true, ButtonDataName: "female", ButtonDataVal: "female", ButtonDataAct: "pub"})
	// act: pub, url, note
	builder.AppendText("Other", TextOption{IsButton: true, ButtonDataName: "other", ButtonDataVal: "other", ButtonDataAct: "url",
		ButtonDataRef: "https://demo.dev/test/action"})

	head, content := builder.Content()
	fmt.Println(head)
	fmt.Printf("%s\n", content)
}
