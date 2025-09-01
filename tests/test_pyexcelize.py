import py_excelize as pex

if __name__ == "__main__":
    wb = pex.open("/home/adhadse/Downloads/test_form_controls_0.xlsx")
    print(wb.sheets())
    controls = wb.list_controls("Sheet 1")
    print(controls)
    for c in controls:
        print(c)
        print(c.type, c.cell, c.linked_cell, c.checked, c.text)
    wb.close()
