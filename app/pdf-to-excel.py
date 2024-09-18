# pdf_to_excel.py
import tabula
import pandas as pd
import sys

def pdf_to_excel(pdf_file_path, excel_file_path):
    try:
        # Read PDF file
        tables = tabula.read_pdf(pdf_file_path, pages='all')
        
        if not tables:
            print("No tables found in the PDF.")
            return
        
        # Write each table to a separate sheet in the Excel file
        with pd.ExcelWriter(excel_file_path) as writer:
            for i, table in enumerate(tables):
                if table is not None and not table.empty:
                    sheet_name = f'Sheet{i+1}'
                    table.to_excel(writer, sheet_name=sheet_name, index=False)
                    print(f"Written table to {sheet_name}")
                else:
                    print(f"Table {i+1} is empty or None and will not be written.")

    except Exception as e:
        print(f"An error occurred: {e}")

# Ensure the script can be run from the command line with arguments
if __name__ == "__main__":
    pdf_file = sys.argv[1]
    excel_file = sys.argv[2]
    pdf_to_excel(pdf_file, excel_file)

