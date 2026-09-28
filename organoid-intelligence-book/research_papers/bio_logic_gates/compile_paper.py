import markdown
import os

def compile_paper():
    input_file = 'Biological_Logic_Gates_Paper.md'
    output_file = 'Final_Biological_Paper.html'
    
    if not os.path.exists(input_file):
        print(f"Error: {input_file} not found.")
        return

    with open(input_file, 'r', encoding='utf-8') as f:
        md_text = f.read()

    # Convert Markdown to HTML
    html_text = markdown.markdown(md_text, extensions=['fenced_code', 'tables'])
    
    # Replace relative image paths with absolute paths so browsers load them correctly
    base_dir = os.path.abspath(os.path.dirname(input_file)).replace('\\', '/')
    html_text = html_text.replace('./results/', f'file:///{base_dir}/results/')

    # Academic HTML Template with MathJax and proper CSS
    full_html = f"""<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <title>Biological Logic Gates Paper</title>
    <!-- MathJax for LaTeX Rendering -->
    <script>
    MathJax = {{
      tex: {{
        inlineMath: [['$', '$'], ['\\\\(', '\\\\)']],
        displayMath: [['$$', '$$'], ['\\\\[', '\\\\]']]
      }},
      svg: {{
        fontCache: 'global'
      }}
    }};
    </script>
    <script type="text/javascript" id="MathJax-script" async
      src="https://cdn.jsdelivr.net/npm/mathjax@3/es5/tex-svg.js">
    </script>
    
    <style>
        body {{
            font-family: "Times New Roman", Times, serif;
            font-size: 12pt;
            line-height: 1.6;
            color: #000;
            max-width: 850px;
            margin: 40px auto;
            padding: 0 20px;
            text-align: justify;
        }}
        h1 {{
            font-size: 20pt;
            text-align: center;
            margin-bottom: 5px;
        }}
        h2 {{
            font-size: 16pt;
            margin-top: 30px;
            margin-bottom: 15px;
        }}
        h3 {{
            font-size: 14pt;
            margin-top: 25px;
            margin-bottom: 10px;
        }}
        p {{
            margin-bottom: 15px;
        }}
        img {{
            max-width: 80%;
            height: auto;
            display: block;
            margin: 25px auto 10px auto;
            border: 1px solid #ccc;
        }}
        /* Fix the EM tag line break issue - caption styling */
        em {{
            font-style: italic;
            display: inline; /* Fixes random line breaks around standard italic text */
        }}
        /* Specifically style paragraphs containing only an em tag as figure captions */
        p > em:only-child {{
            display: block;
            text-align: center;
            font-size: 10pt;
            margin-top: 5px;
            margin-bottom: 25px;
            color: #333;
        }}
        table {{
            width: 100%;
            border-collapse: collapse;
            margin: 20px 0;
        }}
        table, th, td {{
            border: 1px solid black;
        }}
        th, td {{
            padding: 8px;
            text-align: center;
        }}
    </style>
</head>
<body>
    {html_text}
</body>
</html>
"""

    with open(output_file, 'w', encoding='utf-8') as f:
        f.write(full_html)
        
    print(f"Successfully generated {output_file}")

if __name__ == "__main__":
    compile_paper()
