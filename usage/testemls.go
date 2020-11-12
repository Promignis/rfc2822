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
