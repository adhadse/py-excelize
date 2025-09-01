import setuptools
from setuptools import Extension
import os

version = os.environ.get('RELEASE_VERSION', '0.0.0.dev0')
version = version.lstrip('v')
print(f"version: {version}")

setuptools.setup(
    name="py_excelize",
    version=version,
    url="https://github.com/adhadse/pyexcelize",
    author="Anurag Dhadse",
    description="Extract excel form content into structured data.",
    long_description_content_type="text/markdown",
    packages=setuptools.find_packages(include=["py_excelize"]),
    license="MIT",
    platforms="Linux",
    keywords=["go", "golang", "python", "excel", "xlsx", "form", "active-x", "form controls"],
    ext_modules=[],
    py_modules = ["py_excelize.extractor", "py_excelize.utils"],
    package_data={"py_excelize": [
        "*.so",
        "*_go.py",
        "*.py",
        "_*.py",
        "*.h",
        "*.c",
    ]},
    include_package_data=True,
)
