Contributing
=============

Welcome! Contributions are appreciated.

Code of Conduct
---------------

Be respectful and constructive. Focus on improving the project.

How to Contribute
-----------------

1. **Fork the repository**

   .. code-block:: bash

       git clone https://github.com/your-username/oci-prometheus-sd-proxy.git

2. **Create a feature branch**

   .. code-block:: bash

       git checkout -b feature/your-feature

3. **Make changes**
   - Follow existing code style
   - Add tests for new features
   - Update documentation

4. **Test your changes**

   .. code-block:: bash

       make test
       make lint

5. **Push to your fork and create a pull request**

   .. code-block:: bash

       git push origin feature/your-feature

   Then open a PR on GitHub.

Pull Request Guidelines
-----------------------

- Describe what the PR does
- Reference any related issues
- Include tests for new features
- Keep commits focused and descriptive
- Use conventional commit format:
  - ``feat:`` for new features
  - ``fix:`` for bug fixes
  - ``docs:`` for documentation
  - ``test:`` for tests
  - ``chore:`` for maintenance

Code Style
----------

- Follow Go conventions (gofmt, golint)
- Keep functions small and focused
- Write clear variable/function names
- Add comments for complex logic

Testing
-------

All new features must include tests:

.. code-block:: bash

    make test

Reporting Issues
----------------

- Check if issue already exists
- Provide clear reproduction steps
- Include relevant logs
- Describe expected vs actual behavior

Contact
-------

Questions? Email: amaanulhaq.s@outlook.com
