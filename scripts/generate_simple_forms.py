#!/usr/bin/env python3
"""
Generate simple test PDF files with form fields for testing form extraction.
This version uses only the form field types that work reliably with reportlab.
"""

import os
import sys
from datetime import datetime

try:
    from reportlab.pdfgen import canvas
    from reportlab.lib.pagesizes import letter
    from reportlab.lib.colors import black, blue, green, pink, magenta
except ImportError:
    print("Please install reportlab: pip install reportlab")
    sys.exit(1)


def create_basic_form_pdf(output_path):
    """Create a basic PDF with simple form fields."""
    c = canvas.Canvas(output_path, pagesize=letter)
    c.setFont("Helvetica", 12)

    # Title
    c.drawString(50, 750, "Basic Form Example")
    c.line(50, 745, 550, 745)

    # Text Fields
    c.drawString(50, 700, "Name:")
    c.acroForm.textfield(
        name='name',
        tooltip='Enter your name',
        x=150, y=695,
        borderStyle='inset',
        width=300,
        height=20,
        textColor=black,
        fillColor=pink,
        borderColor=black,
        forceBorder=True
    )

    c.drawString(50, 650, "Email:")
    c.acroForm.textfield(
        name='email',
        tooltip='Enter your email',
        x=150, y=645,
        borderStyle='inset',
        width=300,
        height=20,
        textColor=black,
        fillColor=pink,
        borderColor=black,
        forceBorder=True
    )

    # Checkboxes
    c.drawString(50, 600, "Subscribe:")
    c.acroForm.checkbox(
        name='subscribe',
        tooltip='Check to subscribe',
        x=150, y=598,
        buttonStyle='check',
        borderColor=black,
        fillColor=green,
        textColor=black,
        forceBorder=True
    )

    # Radio Buttons
    c.drawString(50, 550, "Gender:")
    c.acroForm.radio(
        name='gender',
        tooltip='Select gender',
        value='male',
        selected=False,
        x=150, y=548,
        buttonStyle='circle',
        borderStyle='solid',
        shape='circle',
        borderColor=black,
        fillColor=magenta,
        textColor=black,
        forceBorder=True
    )
    c.drawString(175, 550, "Male")

    c.acroForm.radio(
        name='gender',
        tooltip='Select gender',
        value='female',
        selected=False,
        x=250, y=548,
        buttonStyle='circle',
        borderStyle='solid',
        shape='circle',
        borderColor=black,
        fillColor=magenta,
        textColor=black,
        forceBorder=True
    )
    c.drawString(275, 550, "Female")

    # Simple Dropdown
    c.drawString(50, 500, "Country:")
    options = [('us', 'United States'), ('ca', 'Canada'), ('uk', 'United Kingdom')]
    c.acroForm.choice(
        name='country',
        tooltip='Select your country',
        value='us',
        x=150, y=495,
        width=200,
        height=20,
        options=options,
        borderColor=black,
        fillColor=pink,
        textColor=black,
        forceBorder=True
    )

    # Note: Buttons are not supported by reportlab's AcroForm
    c.drawString(50, 400, "Note: Submit buttons would go here in a real form")

    c.save()
    print(f"Created: {output_path}")


def create_text_fields_pdf(output_path):
    """Create a PDF focused on different text field types."""
    c = canvas.Canvas(output_path, pagesize=letter)
    c.setFont("Helvetica", 12)

    # Title
    c.drawString(50, 750, "Text Field Examples")
    c.line(50, 745, 550, 745)

    # Regular text field
    c.drawString(50, 700, "Regular Text:")
    c.acroForm.textfield(
        name='regularText',
        tooltip='Regular text field',
        x=200, y=695,
        width=250,
        height=20,
        borderStyle='inset'
    )

    # Required field
    c.drawString(50, 650, "Required Field:")
    c.acroForm.textfield(
        name='requiredField',
        tooltip='This field is required',
        x=200, y=645,
        width=250,
        height=20,
        borderStyle='inset',
        fieldFlags='required'
    )

    # Max length field
    c.drawString(50, 600, "Max 10 chars:")
    c.acroForm.textfield(
        name='maxLengthField',
        tooltip='Maximum 10 characters',
        x=200, y=595,
        width=150,
        height=20,
        borderStyle='inset',
        maxlen=10
    )

    # Multiline text
    c.drawString(50, 550, "Comments:")
    c.acroForm.textfield(
        name='comments',
        tooltip='Multiline text field',
        x=200, y=470,
        width=250,
        height=75,
        borderStyle='inset',
        fieldFlags='multiline'
    )

    # Password field
    c.drawString(50, 430, "Password:")
    c.acroForm.textfield(
        name='password',
        tooltip='Password field',
        x=200, y=425,
        width=250,
        height=20,
        borderStyle='inset',
        fieldFlags='password'
    )

    # Read-only field
    c.drawString(50, 380, "Read-only:")
    c.acroForm.textfield(
        name='readOnly',
        tooltip='Read-only field',
        x=200, y=375,
        width=250,
        height=20,
        borderStyle='inset',
        fieldFlags='readOnly',
        value='This cannot be changed'
    )

    c.save()
    print(f"Created: {output_path}")


def create_choice_fields_pdf(output_path):
    """Create a PDF with different choice field types."""
    c = canvas.Canvas(output_path, pagesize=letter)
    c.setFont("Helvetica", 12)

    # Title
    c.drawString(50, 750, "Choice Field Examples")
    c.line(50, 745, 550, 745)

    # Simple dropdown
    c.drawString(50, 700, "Simple Dropdown:")
    options1 = [('opt1', 'Option 1'), ('opt2', 'Option 2'), ('opt3', 'Option 3')]
    c.acroForm.choice(
        name='simpleDropdown',
        tooltip='Select an option',
        value='opt1',
        x=200, y=695,
        width=200,
        height=20,
        options=options1
    )

    # Dropdown with many options
    c.drawString(50, 650, "State:")
    states = [
        ('', '-- Select State --'),
        ('AL', 'Alabama'), ('AK', 'Alaska'), ('AZ', 'Arizona'),
        ('AR', 'Arkansas'), ('CA', 'California'), ('CO', 'Colorado'),
        ('CT', 'Connecticut'), ('DE', 'Delaware'), ('FL', 'Florida'),
        ('GA', 'Georgia'), ('HI', 'Hawaii'), ('ID', 'Idaho')
    ]
    c.acroForm.choice(
        name='state',
        tooltip='Select your state',
        value='',
        x=200, y=645,
        width=200,
        height=20,
        options=states
    )

    # Radio button groups
    c.drawString(50, 590, "Size:")
    sizes = [('S', 50), ('M', 100), ('L', 150), ('XL', 200)]
    for value, x_offset in sizes:
        c.acroForm.radio(
            name='size',
            tooltip=f'Size {value}',
            value=value.lower(),
            selected=(value == 'M'),  # Default to M
            x=150 + x_offset,
            y=588,
            buttonStyle='circle'
        )
        c.drawString(170 + x_offset, 590, value)

    # Multiple checkboxes (simulating multi-select)
    c.drawString(50, 540, "Features:")
    features = [
        ('feature1', 'Feature 1', 0),
        ('feature2', 'Feature 2', 100),
        ('feature3', 'Feature 3', 200)
    ]
    for name, label, x_offset in features:
        c.acroForm.checkbox(
            name=name,
            tooltip=label,
            x=150 + x_offset,
            y=538,
            buttonStyle='check'
        )
        c.drawString(170 + x_offset, 540, label)

    c.save()
    print(f"Created: {output_path}")


def create_mixed_form_pdf(output_path):
    """Create a PDF with a mix of form fields in a realistic layout."""
    c = canvas.Canvas(output_path, pagesize=letter)
    c.setFont("Helvetica-Bold", 14)

    # Title
    c.drawString(50, 750, "Registration Form")
    c.line(50, 745, 550, 745)

    c.setFont("Helvetica", 11)

    # Personal Information
    c.drawString(50, 720, "First Name:")
    c.acroForm.textfield(
        name='firstName',
        x=130, y=715,
        width=150,
        height=20,
        borderStyle='inset',
        fieldFlags='required'
    )

    c.drawString(300, 720, "Last Name:")
    c.acroForm.textfield(
        name='lastName',
        x=380, y=715,
        width=150,
        height=20,
        borderStyle='inset',
        fieldFlags='required'
    )

    # Email
    c.drawString(50, 680, "Email:")
    c.acroForm.textfield(
        name='emailAddress',
        x=130, y=675,
        width=400,
        height=20,
        borderStyle='inset',
        fieldFlags='required'
    )

    # Age
    c.drawString(50, 640, "Age:")
    c.acroForm.textfield(
        name='age',
        x=130, y=635,
        width=50,
        height=20,
        borderStyle='inset',
        maxlen=3
    )

    # Country dropdown
    c.drawString(200, 640, "Country:")
    countries = [
        ('', '-- Select --'),
        ('us', 'United States'),
        ('ca', 'Canada'),
        ('mx', 'Mexico'),
        ('uk', 'United Kingdom'),
        ('de', 'Germany'),
        ('fr', 'France')
    ]
    c.acroForm.choice(
        name='countrySelect',
        value='',
        x=280, y=635,
        width=150,
        height=20,
        options=countries
    )

    # Newsletter checkbox
    c.drawString(50, 600, "Subscribe to newsletter:")
    c.acroForm.checkbox(
        name='newsletter',
        x=200, y=598,
        buttonStyle='check'
    )

    # Terms checkbox
    c.drawString(50, 570, "I agree to the terms and conditions:")
    c.acroForm.checkbox(
        name='terms',
        x=250, y=568,
        buttonStyle='check',
        fieldFlags='required'
    )

    # Note: Submit button would go here
    c.drawString(50, 500, "Note: Submit button would go here in a real form")

    # Footer
    c.setFont("Helvetica", 8)
    c.drawString(50, 50, f"Form generated on {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")

    c.save()
    print(f"Created: {output_path}")


def main():
    """Generate all test PDF forms."""
    # Create output directory
    output_dir = os.path.join(os.path.dirname(os.path.dirname(__file__)), 'docs', 'test-forms')
    os.makedirs(output_dir, exist_ok=True)

    # Generate test PDFs
    print("Generating test PDF forms...")

    try:
        create_basic_form_pdf(os.path.join(output_dir, 'basic-form.pdf'))
        create_text_fields_pdf(os.path.join(output_dir, 'text-fields.pdf'))
        create_choice_fields_pdf(os.path.join(output_dir, 'choice-fields.pdf'))
        create_mixed_form_pdf(os.path.join(output_dir, 'mixed-form.pdf'))

        print(f"\nAll test forms have been generated in: {output_dir}")
        print("\nGenerated PDFs:")
        print("- basic-form.pdf: Simple form with basic field types")
        print("- text-fields.pdf: Various text field configurations")
        print("- choice-fields.pdf: Dropdowns, radio buttons, and checkboxes")
        print("- mixed-form.pdf: Realistic form with mixed field types")

    except Exception as e:
        print(f"Error generating PDFs: {e}")
        sys.exit(1)


if __name__ == '__main__':
    main()
