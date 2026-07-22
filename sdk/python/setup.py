from setuptools import setup, find_packages

setup(
    name="tender-sdk",
    version="0.1.0",
    description="Python SDK for the Tender AI-native blockchain",
    packages=find_packages(),
    install_requires=["requests"],
)
