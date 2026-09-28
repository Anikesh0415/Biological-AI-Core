import markdown
import os
from xhtml2pdf import pisa

def convert():
    input_file = 'Biological_Logic_Gates_Paper.md'
    output_file = 'Biological_Logic_Gates_Paper.pdf'
    
    if not os.path.exists(input_file):
        print(f"Error: {input_file} not found.")
        return

    with open(input_file, 'r', encoding='utf-8') as f:
        md_text = f.read()

    # Convert Markdown to HTML
    html_text = markdown.markdown(md_text, extensions=['fenced_code', 'tables'])

    # CSS for xhtml2pdf to properly display the layout
    full_html = f"""
    <!DOCTYPE html>
    <html>
    <head>
        <meta charset="utf-8">
        <style>
            @page {{
                size: a4 portrait;
                margin: 2cm;
            }}
            body {{
                font-family: Helvetica, Arial, sans-serif;
                font-size: 11pt;
                line-height: 1.5;
                color: #333;
            }}
            h1, h2, h3 {{
                color: #111;
                border-bottom: 1px solid #ccc;
                padding-bottom: 3px;
                margin-top: 15px;
                margin-bottom: 10px;
            }}
            img {{
                max-width: 500px;
                display: block;
                margin: 15px auto;
            }}
            em {{
                color: #555;
                font-size: 9pt;
                display: block;
                text-align: center;
                margin-bottom: 20px;
            }}
        </style>
    </head>
    <body>
        {html_text}
    </body>
    </html>
    """

    # Create PDF
    with open(output_file, "w+b") as result_file:
        # pisa requires relative links to be resolvable relative to the cwd.
        # Images are in `./results/`, so we can just set dest path.
        pisa_status = pisa.CreatePDF(
            full_html, 
            dest=result_file,
            path=os.path.abspath('.')
        )
        
    if pisa_status.err:
        print("Error during PDF generation")
    else:
        print(f"Successfully generated {output_file}")

if __name__ == "__main__":
    convert()
