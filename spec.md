# Spec

Expected Command to call from client

```
cat [file] | curl -X POST "{ip}/print" -H "username: [username]" -H "teamname: [teamname]" -H "teamid: [teamid]" -H "location: [location]" --data-binary @-
```

When this called execute the following steps:

1. Validate that body characters are valid UTF-8. If not, return 400 Bad Request.
2. Validate that the headers `username`, `teamname`, `teamid`, and `location` are present. If any are missing, return 400 Bad Request.
3. Generate pdf from the body content.
4. Select worker based on (jobID % #workers) where jobID is incremented for each print job and #workers is the number of workers available. The worker list is provided in database.
5. Send the generated pdf to worker via api that is this python code:

```python
from fastapi import FastAPI, status, UploadFile, File, Body
from fastapi.responses import JSONResponse
import os
import subprocess
import tempfile
import time

app = FastAPI()

start_time = time.time()

@app.get("/healthz")
async def health_check():
  uptime_seconds = time.time() - start_time
  env = os.environ.get('PY_ENV', 'development')
  return JSONResponse(
    status_code=status.HTTP_200_OK,
    content={
      "message": "Service is healthy",
      "uptime": uptime_seconds,
      "environment": env
    }
  )

@app.post("/print")
async def print_document(file: UploadFile = File(...)):
  if not file:
    print("No file provided")
    return JSONResponse(
      status_code=status.HTTP_400_BAD_REQUEST,
      content={"message": "No file provided"}
    )

  if file.filename.endswith(".pdf") == False:
    print("Invalid file extension")
    return JSONResponse(
      status_code=status.HTTP_400_BAD_REQUEST,
      content={"message": "Invalid file extension. Only .pdf files are accepted."}
    )

  try:
    tmp_dir = os.path.join(os.getcwd(), "tmp")
    os.makedirs(tmp_dir, exist_ok=True)

    with tempfile.NamedTemporaryFile(dir=tmp_dir, suffix=f"_{file.filename}", delete=False) as temp_file:
      content = await file.read()
      temp_file.write(content)
      temp_file_path = temp_file.name

    if os.environ.get('PY_ENV') == 'production':
      result = subprocess.run(["lpr", temp_file_path], capture_output=True, text=True)
      if result.returncode != 0:
        raise Exception(f"Print command failed: {result.stderr}")
    else:
      print("This is a development environment. Skipping actual print command...")
    print(f"File {file.filename} sent to printer")
  except Exception as e:
    print("Failed to print the document:", str(e))
    return JSONResponse(
      status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
      content={"message": f"Failed to print the document: {str(e)}"}
    )

  return JSONResponse(
    status_code=status.HTTP_200_OK,
    content={"message": f"File {file.filename} sent to printer successfully"}
  )

@app.post("/print/manual")
async def manual_print(data: dict = Body(...)):
  filename = data.get("filename")
  if not filename:
    return JSONResponse(
      status_code=status.HTTP_400_BAD_REQUEST,
      content={"message": "filename is required in JSON body"}
    )
  try:
    tmp_dir = os.path.join(os.getcwd(), "tmp")

    matching_files = [f for f in os.listdir(tmp_dir) if f.endswith(f"_{filename}")]

    if not matching_files:
      return JSONResponse(
        status_code=status.HTTP_404_NOT_FOUND,
        content={"message": f"No file found ending with '{filename}' in tmp directory"}
      )

    if len(matching_files) > 1:
      return JSONResponse(
        status_code=status.HTTP_400_BAD_REQUEST,
        content={"message": f"Multiple files found ending with '{filename}': {matching_files}"}
      )

    file_path = os.path.join(tmp_dir, matching_files[0])

    if os.environ.get('PY_ENV') == 'production':
      result = subprocess.run(["lpr", file_path], capture_output=True, text=True)
      if result.returncode != 0:
        raise Exception(f"Print command failed: {result.stderr}")
    else:
      print("This is a development environment. Skipping actual print command...")

    print(f"Manually printed file: {matching_files[0]}")
    return JSONResponse(
      status_code=status.HTTP_200_OK,
      content={"message": f"File {filename} sent to printer successfully"}
    )

  except Exception as e:
    print("Failed to manually print the document:", str(e))
    return JSONResponse(
      status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
      content={"message": f"Failed to manually print the document: {str(e)}"}
    )
```

(using `/print` endpoint) with the generated pdf as a file upload. If the worker returns an error, return 500 Internal Server Error to the client.

## PDF generation

use this libarary `go get github.com/johnfercher/maroto/v2@v2.4.0`

but i have legacy code here in ts as an example:

```ts
import { type NextRequest, NextResponse } from "next/server";
import { jsPDF } from "jspdf";
import { ubuntuSansMono } from "@/constants/UbuntuMono";
import { parseCode } from "@/utils/ParseCode";

interface IGeneratePdfRequestBody {
  code: string;
  codeHeader: string;
}

export async function POST(request: NextRequest) {
  const { code, codeHeader } = await request.json() as IGeneratePdfRequestBody;
  if(!code || !codeHeader) {
    return new NextResponse("Invalid request body.", { status: 400 });
  }

  const pdf = new jsPDF();
  pdf.setFontSize(10);
  pdf.addFileToVFS("UbuntuSansMono.ttf", ubuntuSansMono);
  pdf.addFont("UbuntuSansMono.ttf", "UbuntuSansMono", "normal");
  pdf.setFont("UbuntuSansMono");

  const lineHeight = 5;
  const pageHeight = pdf.internal.pageSize.height - 10;
  const pageWidth = pdf.internal.pageSize.width;
  const margin = 10;
  const maxLineWidth = pageWidth - 2 * margin;
  let y = margin;

  const addHeader = (pageNumber: number, totalPages: number) => {
    const headerHeight = lineHeight * 2;
    pdf.setFillColor(255, 255, 255);
    pdf.rect(
      margin,
      margin - lineHeight,
      pageWidth - 2 * margin,
      headerHeight,
      "F"
    );

    pdf.text(codeHeader, margin, margin);
    pdf.text(
      `Page ${pageNumber} / ${totalPages}`,
      pageWidth - margin - 24,
      margin
    );
    y = margin + lineHeight * 2;
  };

  const updatePageNumber = (pageNumber: number, totalPages: number) => {
    pdf.setPage(pageNumber);
    pdf.setFillColor(255, 255, 255);
    pdf.rect(pageWidth - margin - 30, margin - 3, 30, 6, "F");
    pdf.text(
      `Page ${pageNumber} / ${totalPages}`,
      pageWidth - margin - 24,
      margin
    );
  };

  const lineNumberWidth = (pdf.getStringUnitWidth("0000") * 12) / pdf.internal.scaleFactor;

  const parsedCode = parseCode(code);

  let pageNumber = 1;
  let totalPages = 1;

  addHeader(1, 1);

  for(let i=0; i<parsedCode.length; ++i) {
    const lines = pdf.splitTextToSize(parsedCode[i]!, maxLineWidth - lineNumberWidth) as string[];
    for (let j=0; j<lines.length; ++j) {
      if (y + lineHeight > pageHeight) {
        pdf.addPage();
        y = margin;
        pageNumber++;
        totalPages++;
        addHeader(pageNumber, totalPages);
      }
      if (j === 0) {
        pdf.text(`${i + 1}`, margin, y);
      }
      pdf.text(lines[j]!, margin + lineNumberWidth, y);
      y += lineHeight;
    }
  }

  for (let i = 1; i <= totalPages; i++) {
    updatePageNumber(i, totalPages);
  }


  const blob = new Blob([pdf.output("blob")], { type: "application/pdf" });

  return new NextResponse(blob, {
    headers: {
      "Content-Type": "application/pdf",
    },
  });
}
```

```ts
// ParseCode.ts
export function parseCode(input: string): string[] {
  const modifiedInput = input.replace(/\t/g, "    ").replace(/\n$/, "");
  return modifiedInput.split(/(?<!\\)\n/);
}
```

```ts
// entry point
import { type NextRequest, NextResponse } from "next/server";
import { getPages } from "@/utils/PrintingUtils";
import axios from "axios";
import { createHeader } from "@/utils/CodeHeader";
import type { IQuotaResponse } from "@/types/GET";
import type { UploadResponse } from "@/types/POST";

interface IPrintRequestBody {
  code: string;
}

export async function POST(request: NextRequest) {
  const teamName = request.headers.get("X-DOMjudge-Name");
  const loginUsername = request.headers.get("X-DOMjudge-Login");
  const loginPassword = request.headers.get("X-DOMjudge-Pass");
  if(!teamName || !loginUsername || !loginPassword) {
    return new NextResponse("No credentials", { status: 400 });
  }

  const body = await request.json() as IPrintRequestBody;

  const remainingQuotaResponse = await axios.get<IQuotaResponse>(`${process.env.NEXT_PUBLIC_API_URL}/api/printing/quota`, {
    headers: {
      "X-DOMjudge-Name": teamName,
      "X-DOMjudge-Login": loginUsername,
      "X-DOMjudge-Pass": loginPassword,
    }
  });
  const remainingQuota = remainingQuotaResponse.data.quota;

  const usingPages = getPages(body.code);
  if(usingPages > remainingQuota) {
    return new NextResponse(`Insufficient printing quota. You have ${remainingQuota} pages remaining, but your print job requires ${usingPages} pages.`, { status: 400 });
  }

  try {
    const codeHeader = createHeader(teamName, loginUsername);
    const pdfBlob = await fetch(`${process.env.NEXT_PUBLIC_SITE_URL}/api/pdf`, {
      body: JSON.stringify({
        code: body.code,
        codeHeader,
      }),
      method: "POST",
    }).then((r) => r.blob());

    const formData = new FormData();
    formData.append("file", pdfBlob, `print_job_${Date.now()}_${teamName}.pdf`);
    formData.append("pages", usingPages.toString());

    const uploadResponse = await axios.post<UploadResponse>(`${process.env.NEXT_PUBLIC_API_URL}/api/printing/print`, formData, {
      headers: {
        "X-DOMjudge-Login": loginUsername,
        "X-DOMjudge-Pass": loginPassword,
      }
    });
    return NextResponse.json(uploadResponse.data);
  } catch (e: unknown) {
    console.error(e);
    return new NextResponse("Failed to process print job.", { status: 500 });
  }
}
```

## Schema

This is just example from old project!

```go
package models

import (
	"time"
)

type Transaction struct {
	ID        uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	Username  string     `json:"username" gorm:"not null"`
	FileName  string     `json:"file_name" gorm:"not null"`
	FilePath  string     `json:"file_path" gorm:"not null"`
	Pages     int        `json:"pages" gorm:"not null"`
	WorkerID  uint       `json:"worker_id" gorm:"not null"`
	Worker    Worker     `json:"worker" gorm:"foreignKey:WorkerID"`
	Status    string     `json:"status" gorm:"not null;default:'pending'"`
	PrintedAt *time.Time `json:"printed_at"`
}
```

```go
package models

type Worker struct {
	ID			 uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	IPAddress string `json:"ip_address" gorm:"not null"`
}
```
