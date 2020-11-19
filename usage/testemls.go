package main

var TestEml2 = []byte(`Delivered-To: efgh@promignis.com
Received: by 2002:a02:c891:0:0:0:0:0 with SMTP id m17csp2767835jao;
		Sun, 25 Oct 2020 18:04:21 -0700 (PDT)
ARC-Seal: i=1; a=rsa-sha256; t=1603946533; cv=none;
		d=google.com; s=arc-20160816;
		b=BVpkSaAhCR7Gqg2FGocH72r0OLPeHoW+nbFRxTj07hQTbB6kwZo83BvkKi8XCb5shj
		/3O+jE3G9BFE05J3yR7ixFDjaDwmVN/54uyUZmTjONQixcwFK8P1TinfFPncQkeqnQJX
		kpByMrC9zWP1JRGQb9gcViVsgHZdfLFsXxwMJvdXqWpxIIY8tvRIFoXuZkt+wYlUGneL
		dGqsnYu88QhvjFAjPRhGMZBP1TCf/noWFDNBp1u//mvC9skMLYwIuk37t2elV5M0uZ8F
		w7luRzTLrxOwHR79cX0uWO5uswBeWojmWR8DRpEIOO4/Vm/EP2PkSYvvnYPH7+6bHd5P
		RIXQ==
MIME-Version: 1.0
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
Hi this is the body

--sub1
Content-Type: text/html; charset="UTF-8"

<div dir="ltr">Hi this is the body<div><br></div></div>
<div dir="ltr">Hi this is the body<div><br></div></div>

--sub1--
--main1
Content-Type: text/plain; charset="US-ASCII"; name="attac.txt"
Content-Disposition: attachment; filename="attac.txt"
Content-Transfer-Encoding: base64
Content-ID: <f_kgruatpx0>
X-Attachment-Id: f_kgruatpx0

U2FtcGxlVGV4dCBkYXRhIGhlcmUg
U2FtcGxlVGV4dCBkYXRhIGhlcmUg
U2FtcGxlVGV4dCBkYXRhIGhlcmUg
U2FtcGxlVGV4dCBkYXRhIGhlcmUg
--main1--
`)

var onlyHeaders = []byte("MIME-Version: 1.0\nFrom: revant jha <abc.94@gmail.com>\r\nDate: Tue, 27 Oct 2020 16:11:25 +0530\r\nMessage-ID: <CALa9RR=0AnAvVYBN_XeuZ+z51M7Em-i_RoYC3Ur8WmEt4h+mig@mail.gmail.com>\r\nSubject: test eml\r\nTo: efgh@promignis.com\n")

var TestEml3 = []byte(`Delivered-To: efgh@promignis.com
Received: by 2002:a02:c891:0:0:0:0:0 with SMTP id m17csp2767835jao;
		Sun, 25 Oct 2020 18:04:21 -0700 (PDT)
MIME-Version: 1.0
From: revant jha <abc.94@gmail.com>
Date: Tue, 27 Oct 2020 16:11:25 +0530
Message-ID: <CALa9RR=0AnAvVYBN_XeuZ+z51M7Em-i_RoYC3Ur8WmEt4h+mig@mail.gmail.com>
Subject: test eml
To: efgh@promignis.com
Content-Type: text/plain; charset="UTF-8"

Hi this is the body

`)

var encodedEml = []byte(`MIME-Version: 1.0
Date: Thu, 19 Nov 2020 14:05:07 +0530
Message-ID: <CALa9RRnWzEZxi1GEMnAxHYX=JqPNNef2UzJSdgVsuY46zanc=w@mail.gmail.com>
Subject: sometest
From: revant jha <revantjha.94@gmail.com>
To: revant@promignis.com
Content-Type: multipart/alternative; boundary="base"

--base
Content-Type: text/plain; charset="UTF-8"
Content-Transfer-Encoding: base64

4oiuIEXii4VkYSA9IFEsICBuIOKGkiDiiJ4sIOKIkSBmKGkpID0g4oiPIGcoaSksIOKIgHjiiIji
hJ06IOKMiHjijIkgPSDiiJLijIriiJJ44oyLLCDOsSDiiKcgwqzOsiA9IMKsKMKszrEg4oioIM6y
KSwNCg0KICDihJUg4oqGIOKEleKCgCDiioIg4oSkIOKKgiDihJog4oqCIOKEnSDiioIg4oSCLCDi
iqUgPCBhIOKJoCBiIOKJoSBjIOKJpCBkIOKJqiDiiqQg4oeSIChBIOKHlCBCKSwNCg0KICAySOKC
giArIE/igoIg4oeMIDJI4oKCTywgUiA9IDQuNyBrzqksIOKMgCAyMDAgbW0NCg==
--base
Content-Type: text/html; charset="UTF-8"
Content-Transfer-Encoding: base64

PGRpdiBkaXI9Imx0ciI+PHByZSBzdHlsZT0iY29sb3I6cmdiKDAsMCwwKSI+4oiuIEXii4VkYSA9
IFEsICBuIOKGkiDiiJ4sIOKIkSBmKGkpID0g4oiPIGcoaSksIOKIgHjiiIjihJ06IOKMiHjijIkg
PSDiiJLijIriiJJ44oyLLCDOsSDiiKcgwqzOsiA9IMKsKMKszrEg4oioIM6yKSwNCg0KICDihJUg
4oqGIOKEleKCgCDiioIg4oSkIOKKgiDihJog4oqCIOKEnSDiioIg4oSCLCDiiqUgJmx0OyBhIOKJ
oCBiIOKJoSBjIOKJpCBkIOKJqiDiiqQg4oeSIChBIOKHlCBCKSwNCg0KICAySOKCgiArIE/igoIg
4oeMIDJI4oKCTywgUiA9IDQuNyBrzqksIOKMgCAyMDAgbW08L3ByZT48L2Rpdj4NCg==
--base--`)
