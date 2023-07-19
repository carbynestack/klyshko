/*
 * Copyright (c) 2023 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
 *
 * SPDX-License-Identifier: Apache-2.0
 */
#include "Tools/ezOptionParser.h"

#include "Math/Setup.h"
#include "Math/field_types.h"
#include "Math/gfp.hpp"
#include "Math/gf2n.h"
#include "Machines/SPDZ.hpp"
#include "Protocols/CowGearOptions.h"
#include "Protocols/CowGearShare.h"
#include "Protocols/CowGearPrep.hpp"
#include "Tools/Buffer.h"
#include <fstream>
#include <assert.h>
#include <boost/filesystem.hpp>

/**
 * Outputs the specified number of tuples of the given type into a file in the given directory.
 *
 * @param preprocessing The preprocessing implementation.
 * @param tuple_type The tuple type to be generated.
 * @param tuple_count The number of tuples to be generated.
 * @param tuple A container for storing the elements of a tuple.
 * @param names The network setup.
 */
template <class T, std::size_t ELEMENTS>
void generate_tuples(Preprocessing<T> &preprocessing, Dtype tuple_type, int tuple_count, array<T, ELEMENTS> &tuple, Names& names)
{

    string filename = Sub_Data_Files<T>::get_filename(names, tuple_type);
    ofstream out(filename, ios::out | ios::binary);
    file_signature<T>().output(out);

    // Generate and output the tuples
    std::cout << "Generating " << tuple_count << " tuples of type " << DataPositions::dtype_names[tuple_type] << std::endl;
    for (int i = 0; i < tuple_count; i++)
    {
        preprocessing.get(tuple_type, tuple.data());
        for (const T &t : tuple)
        {
            t.output(out, false);
        }
    }

    std::cout << "Wrote " << tuple_count << " tuples of type " << DataPositions::dtype_names[tuple_type] << " to " << filename << std::endl;
    out.close();
}

template <class T>
void generate_tuples(int player_id, int number_of_players, int port, int tuple_count, string tuple_type, string playerfile)
{
    // Create working directory, if it doesn't exist yet
    std::string working_dir = get_prep_sub_dir<T>(PREP_DIR, number_of_players);
    boost::filesystem::path path(working_dir);
    if (!(boost::filesystem::exists(path)))
    {
        if (boost::filesystem::create_directory(path))
            std::cout << "Non-existing working directory " << path << " created" << std::endl;
    }

    // Init networking
    Names N;
    N.init(player_id, port, playerfile, number_of_players);

    PlainPlayer P(N);

    T::clear::template write_setup<T>(number_of_players);

    // Initialize MAC key, if not existing
    auto mac_key = read_generate_write_mac_key<T>(P);

    // Required for keeping track of preprocessing material
    DataPositions usage(P.num_players());

    // Configure the player to use given MAC key
    typename T::MAC_Check output(mac_key);
    output.setup(P);

    // Initialize preprocessing
    CowGearPrep<T> preprocessing(0, usage);
    SubProcessor<T> processor(output, preprocessing, P);

    // Generate requested tuples
    if (tuple_type == "bits")
    {
        array<T, 1> bit;
        generate_tuples<T, 1>(preprocessing, DATA_BIT, tuple_count, bit, N);
    }
    else if (tuple_type == "squares")
    {
        array<T, 2> square;
        generate_tuples<T, 2>(preprocessing, DATA_SQUARE, tuple_count, square, N);
    }
    else if (tuple_type == "inverses")
    {
        array<T, 2> inverse;
        generate_tuples<T, 2>(preprocessing, DATA_INVERSE, tuple_count, inverse, N);
    }
    else if (tuple_type == "triples")
    {
        array<T, 3> triple;
        generate_tuples<T, 3>(preprocessing, DATA_TRIPLE, tuple_count, triple, N);
    }
    else
    {
        std::cerr << "Tuple type not supported: " << tuple_type << std::endl;
        exit(EXIT_FAILURE);
    }

    // Perform the MAC check
    output.Check(P);
}

int main(int argc, const char **argv)
{
    // Define and parse command line parameters
    ez::ezOptionParser opt;
    CowGearOptions(opt, argc, argv);
    opt.add(
        "",                   // Default value
        1,                    // Required?
        1,                    // Number of values expected
        0,                    // Delimiter, if expecting multiple args
        "Number of parties",  // Help description
        "-N",                 // Short name
        "--number-of-parties" // Long name
    );
    opt.add(
        "",                                                  // Default value
        1,                                                   // Required?
        1,                                                   // Number of values expected
        0,                                                   // Delimiter, if expecting multiple args
        "This player's number, starting with 0 (required).", // Help description
        "-p",                                                // Short name
        "--player"                                           // Long name
    );
    opt.add(
        "players",                                              // Default value
        0,                                                      // Required?
        1,                                                      // Number of values expected
        0,                                                      // Delimiter, if expecting multiple args
        "Playerfile containing host:port information per line", // Help description
        "-pf",                                                  // Short name
        "--playerfile"                                          // Long name
    );
    opt.add(
        "",                                        // Default value
        1,                                         // Required?
        1,                                         // Number of values expected
        0,                                         // Delimiter, if expecting multiple args
        "The field type to use. One of gfp, gf2n", // Help description
        "-ft",                                     // Short name
        "--field-type"                             // Long name
    );
    opt.add(
        "",                     // Default
        0,                      // Required?
        1,                      // Number of args expected
        0,                      // Delimiter if expecting multiple args
        "Prime for gfp field",  // Help description
        "-pr",                  // Short name
        "--prime"               // Long name
    );
    opt.add(
        "",                                                                    // Default value
        1,                                                                     // Required?
        1,                                                                     // Number of values expected
        0,                                                                     // Delimiter, if expecting multiple args
        "Tuple type to be generated. One of bits, inverses, squares, triples", // Help description
        "-tt",                                                                 // Short name
        "--tuple-type"                                                         // Long name
    );
    opt.add(
        "5000",                              // Default value
        0,                                   // Required?
        1,                                   // Number of values expected
        0,                                   // Delimiter, if expecting multiple args
        "Local port number (default: 5000)", // Help description
        "-P",                                // Short name
        "--port"                             // Long name
    );
    opt.add(
        "100000",                                         // Default value
        0,                                                // Required?
        1,                                                // Number of values expected
        0,                                                // Delimiter, if expecting multiple args
        "Number of tuples to generate (default: 100000)", // Help description
        "-tc",                                            // Short name
        "--tuple-count"                                   // Long name
    );
    opt.parse(argc, argv);
    if (!opt.isSet("-N") || !opt.isSet("-p") || !opt.isSet("-ft") || !opt.isSet("-tt"))
    {
        string usage;
        opt.getUsage(usage);
        cout << usage;
        exit(0);
    }

    int player_id, number_of_players, port, tuple_count;
    string field_type, tuple_type, playerfile;
    opt.get("-p")->getInt(player_id);
    opt.get("-N")->getInt(number_of_players);
    opt.get("-P")->getInt(port);
    opt.get("-ft")->getString(field_type);
    opt.get("-tt")->getString(tuple_type);
    opt.get("-tc")->getInt(tuple_count);
    opt.get("-pf")->getString(playerfile);

    if (mkdir_p(PREP_DIR) == -1)
    {
        throw runtime_error(
            (string) "cannot use " + PREP_DIR + " (set another PREP_DIR in CONFIG when building if needed)");
    }

    if (field_type == "gfp")
    {

        // Read field-specific paramaters
        string prime;
        if (!opt.isSet("--prime")) {
            std::cerr << "No prime given for gfp" << std::endl;
            exit(EXIT_FAILURE);
        }
        opt.get("--prime")->getString(prime);
        std::cout << "Using prime '" << prime << "'" << std::endl;

        // Compute number of 64-bit words needed
        const int prime_length = 128;
        const int n_limbs = (prime_length + 63) / 64;

        // Initialize field (first argument is counter; convention is 0 for online, 1 for offline phase)
        typedef gfp_<1, n_limbs> F;
        F::init_field(prime, true);

        // Define share type
        typedef CowGearShare<F> T;

        generate_tuples<T>(player_id, number_of_players, port, tuple_count, tuple_type, playerfile);
    }
    else if (field_type == "gf2n")
    {
        // Define share type
        typedef CowGearShare<gf2n_short> T;

        // Initialize field
        gf2n_short::init_field(40);

        generate_tuples<T>(player_id, number_of_players, port, tuple_count, tuple_type, playerfile);
    } else {
        std::cerr << "Field type not supported: " << field_type << std::endl;
        exit(EXIT_FAILURE);
    }

}
