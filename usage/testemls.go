package main

var TestEml2 = []byte(`MIME-Version: 1.0
From: revant jha <abc.94@gmail.com>
Date: Tue, 27 Oct 2020 16:11:25 +0530
Message-ID: <CALa9RR=0AnAvVYBN_XeuZ+z51M7Em-i_RoYC3Ur8WmEt4h+mig@mail.gmail.com>
Subject: test eml
To: efgh@promignis.com
Content-Type: multipart/mixed; boundary="main1"

--main1
Content-Type: multipart/alternative; boundary="sub1"

--sub1
Content-Type: text/plain; charset="UTF-8"

Hi this is the body

--sub1
Content-Type: text/html; charset="UTF-8"

<div dir="ltr">Hi this is the body<div><br></div></div>

--sub1--
--main1
Content-Type: text/plain; charset="US-ASCII"; name="attac.txt"
Content-Disposition: attachment; filename="attac.txt"
Content-Transfer-Encoding: base64
Content-ID: <f_kgruatpx0>
X-Attachment-Id: f_kgruatpx0

U2FtcGxlVGV4dCBkYXRhIGhlcmUg
--main1--
`)
