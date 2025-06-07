Module RehabManager
    Sub Main()
        Console.WriteLine("Welcome to Rehab Manager")
        Dim running As Boolean = True
        Dim tasks As New List(Of String)()

        While running
            Console.WriteLine("1. Add task")
            Console.WriteLine("2. View tasks")
            Console.WriteLine("3. Exit")
            Dim choice As String = Console.ReadLine()

            Select Case choice
                Case "1"
                    Console.Write("Enter task: ")
                    tasks.Add(Console.ReadLine())
                Case "2"
                    Console.WriteLine("Tasks:")
                    For Each t In tasks
                        Console.WriteLine("- " & t)
                    Next
                Case "3"
                    running = False
                Case Else
                    Console.WriteLine("Invalid option.")
            End Select
        End While

        Console.WriteLine("Goodbye!")
    End Sub
End Module
