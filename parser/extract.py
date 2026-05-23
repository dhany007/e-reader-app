import sys
import json
import fitz  # PyMuPDF


def extract_pages(pdf_path: str) -> list:
    doc = fitz.open(pdf_path)
    pages = []
    for i, page in enumerate(doc):
        text = page.get_text("text").strip()
        pages.append({"page": i + 1, "text": text})
    doc.close()
    return pages


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print(json.dumps({"error": "usage: extract.py <pdf_path>"}), file=sys.stderr)
        sys.exit(1)

    try:
        result = extract_pages(sys.argv[1])
        print(json.dumps(result))
    except Exception as e:
        print(json.dumps({"error": str(e)}), file=sys.stderr)
        sys.exit(1)
